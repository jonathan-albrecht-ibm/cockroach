// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package server

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/kv/kvserver/liveness/livenesspb"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/server/serverpb"
	"github.com/cockroachdb/cockroach/pkg/util"
	"github.com/gorilla/mux"
)

type nodeStatus struct {
	// Fields that are a subset of NodeDescriptor.
	NodeID        roachpb.NodeID      `json:"node_id"`
	Address       util.UnresolvedAddr `json:"address"`
	Attrs         roachpb.Attributes  `json:"attrs"`
	Locality      roachpb.Locality    `json:"locality"`
	ServerVersion roachpb.Version     `json:"ServerVersion"`
	BuildTag      string              `json:"build_tag"`
	StartedAt     int64               `json:"started_at"`
	ClusterName   string              `json:"cluster_name"`
	SQLAddress    util.UnresolvedAddr `json:"sql_address"`

	// Other fields that are a subset of roachpb.NodeStatus.
	Metrics           map[string]float64 `json:"metrics,omitempty"`
	TotalSystemMemory int64              `json:"total_system_memory,omitempty"`
	NumCpus           int32              `json:"num_cpus,omitempty"`
	UpdatedAt         int64              `json:"updated_at,omitempty"`

	// Retrieved from the liveness status map.
	LivenessStatus livenesspb.NodeLivenessStatus `json:"liveness_status"`
}

// Response struct for listNodes.
//
// swagger:model nodesResponse
type nodesResponse struct {
	// swagger:allOf
	Nodes []nodeStatus `json:"nodes"`
	// Continuation offset for the next paginated call, if more values are present.
	// Specify as the `offset` parameter.
	Next int `json:"next,omitempty"`
}

// swagger:operation GET /nodes/ listNodes
//
// List nodes
//
// List all nodes on this cluster.
//
// Client must be logged-in as a user with admin privileges.
//
// ---
// parameters:
// - name: limit
//   type: integer
//   in: query
//   description: Maximum number of results to return in this call.
//   required: false
// - name: offset
//   type: integer
//   in: query
//   description: Continuation offset for results after a past limited run.
//   required: false
// produces:
// - application/json
// security:
// - api_session: []
// responses:
//   "200":
//     description: List nodes response.
//     schema:
//       "$ref": "#/definitions/nodesResponse"
func (a *apiV2Server) listNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, offset := getSimplePaginationValues(r)
	ctx = apiToOutgoingGatewayCtx(ctx, r)

	nodes, next, err := a.status.nodesHelper(ctx, limit, offset)
	if err != nil {
		apiV2InternalError(ctx, err, w)
		return
	}
	var resp nodesResponse
	resp.Next = next
	for _, n := range nodes.Nodes {
		resp.Nodes = append(resp.Nodes, nodeStatus{
			NodeID:            n.Desc.NodeID,
			Address:           n.Desc.Address,
			Attrs:             n.Desc.Attrs,
			Locality:          n.Desc.Locality,
			ServerVersion:     n.Desc.ServerVersion,
			BuildTag:          n.Desc.BuildTag,
			StartedAt:         n.Desc.StartedAt,
			ClusterName:       n.Desc.ClusterName,
			SQLAddress:        n.Desc.SQLAddress,
			Metrics:           n.Metrics,
			TotalSystemMemory: n.TotalSystemMemory,
			NumCpus:           n.NumCpus,
			UpdatedAt:         n.UpdatedAt,
			LivenessStatus:    nodes.LivenessByNodeID[n.Desc.NodeID],
		})
	}
	writeJSONResponse(ctx, w, 200, resp)
}

func parseRangeIDs(input string, w http.ResponseWriter) (ranges []roachpb.RangeID, ok bool) {
	if len(input) == 0 {
		return nil, true
	}
	for _, reqRange := range strings.Split(input, ",") {
		rangeID, err := strconv.ParseInt(reqRange, 10, 64)
		if err != nil {
			http.Error(w, "invalid range ID", http.StatusBadRequest)
			return nil, false
		}

		ranges = append(ranges, roachpb.RangeID(rangeID))
	}
	return ranges, true
}

type nodeRangeResponse struct {
	// swagger:allOf
	RangeInfo rangeInfo `json:"range_info"`
	Error     string    `json:"error,omitempty"`
}

// swagger:model rangeResponse
type rangeResponse struct {
	// swagger:allOf
	Responses map[string]nodeRangeResponse `json:"responses_by_node_id"`
}

// swagger:operation GET /ranges/{range_id}/ listRange
//
// Get info about a range
//
// Retrieves more information about a specific range.
//
// Client must be logged-in as a user with admin privileges.
//
// ---
// parameters:
// - name: range_id
//   in: path
//   type: integer
//   required: true
// produces:
// - application/json
// security:
// - api_session: []
// responses:
//   "200":
//     description: List range response
//     schema:
//       "$ref": "#/definitions/rangeResponse"
func (a *apiV2Server) listRange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = apiToOutgoingGatewayCtx(ctx, r)
	vars := mux.Vars(r)
	rangeID, err := strconv.ParseInt(vars["range_id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid range ID", http.StatusBadRequest)
		return
	}

	response := &rangeResponse{
		Responses: make(map[string]nodeRangeResponse),
	}

	rangesRequest := &serverpb.RangesRequest{
		RangeIDs: []roachpb.RangeID{roachpb.RangeID(rangeID)},
	}

	dialFn := func(ctx context.Context, nodeID roachpb.NodeID) (interface{}, error) {
		client, err := a.status.dialNode(ctx, nodeID)
		return client, err
	}
	nodeFn := func(ctx context.Context, client interface{}, _ roachpb.NodeID) (interface{}, error) {
		status := client.(serverpb.StatusClient)
		return status.Ranges(ctx, rangesRequest)
	}
	responseFn := func(nodeID roachpb.NodeID, resp interface{}) {
		rangesResp := resp.(*serverpb.RangesResponse)
		// Age the MVCCStats to a consistent current timestamp. An age that is
		// not up to date is less useful.
		if len(rangesResp.Ranges) == 0 {
			return
		}
		var ri rangeInfo
		ri.init(rangesResp.Ranges[0])
		response.Responses[nodeID.String()] = nodeRangeResponse{RangeInfo: ri}
	}
	errorFn := func(nodeID roachpb.NodeID, err error) {
		response.Responses[nodeID.String()] = nodeRangeResponse{
			Error: err.Error(),
		}
	}

	if err := a.status.iterateNodes(
		ctx, fmt.Sprintf("details about range %d", rangeID), dialFn, nodeFn, responseFn, errorFn,
	); err != nil {
		apiV2InternalError(ctx, err, w)
		return
	}
	writeJSONResponse(ctx, w, 200, response)
}

// rangeDescriptorInfo contains a subset of fields from roachpb.RangeDescriptor
// that are safe to be returned from APIs.
type rangeDescriptorInfo struct {
	RangeID  roachpb.RangeID `json:"range_id"`
	StartKey roachpb.RKey    `json:"start_key,omitempty"`
	EndKey   roachpb.RKey    `json:"end_key,omitempty"`

	// Set for HotRanges.
	StoreID          roachpb.StoreID `json:"store_id"`
	QueriesPerSecond float64         `json:"queries_per_second"`
}

func (r *rangeDescriptorInfo) init(rd *roachpb.RangeDescriptor) {
	if rd == nil {
		*r = rangeDescriptorInfo{}
		return
	}
	*r = rangeDescriptorInfo{
		RangeID:  rd.RangeID,
		StartKey: rd.StartKey,
		EndKey:   rd.EndKey,
	}
}

type rangeInfo struct {
	// swagger:allOf
	Desc rangeDescriptorInfo `json:"desc"`

	// Subset of fields copied from serverpb.RangeInfo
	Span          serverpb.PrettySpan      `json:"span"`
	SourceNodeID  roachpb.NodeID           `json:"source_node_id,omitempty"`
	SourceStoreID roachpb.StoreID          `json:"source_store_id,omitempty"`
	ErrorMessage  string                   `json:"error_message,omitempty"`
	LeaseHistory  []roachpb.Lease          `json:"lease_history"`
	Problems      serverpb.RangeProblems   `json:"problems"`
	Stats         serverpb.RangeStatistics `json:"stats"`
	Quiescent     bool                     `json:"quiescent,omitempty"`
	Ticking       bool                     `json:"ticking,omitempty"`
}

func (ri *rangeInfo) init(r serverpb.RangeInfo) {
	*ri = rangeInfo{
		Span:          r.Span,
		SourceNodeID:  r.SourceNodeID,
		SourceStoreID: r.SourceStoreID,
		ErrorMessage:  r.ErrorMessage,
		LeaseHistory:  r.LeaseHistory,
		Problems:      r.Problems,
		Stats:         r.Stats,
		Quiescent:     r.Quiescent,
		Ticking:       r.Ticking,
	}
	ri.Desc.init(r.State.Desc)
}

// Response struct for listNodeRanges.
//
// swagger:model nodeRangesResponse
type nodeRangesResponse struct {
	Ranges []rangeInfo `json:"ranges"`
	Next   int         `json:"next,omitempty"`
}

// swagger:operation GET /nodes/{node_id}/ranges/ listNodeRanges
//
// List ranges on a node
//
// Lists information about ranges on a specified node. If a list of range IDs
// is specified, only information about those ranges is returned.
//
// Client must be logged-in as a user with admin privileges.
//
// ---
// parameters:
// - name: node_id
//   in: path
//   type: integer
//   description: ID of node to query, or `local` for local node.
//   required: true
// - name: ranges
//   in: query
//   type: array
//   required: false
//   description: IDs of ranges to return information for. All ranges returned
//     if unspecified.
//   items:
//     type: integer
// - name: limit
//   type: integer
//   in: query
//   description: Maximum number of results to return in this call.
//   required: false
// - name: offset
//   type: integer
//   in: query
//   description: Continuation offset for results after a past limited run.
//   required: false
// produces:
// - application/json
// security:
// - api_session: []
// responses:
//   "200":
//     description: Node ranges response.
//     schema:
//       "$ref": "#/definitions/nodeRangesResponse"
func (a *apiV2Server) listNodeRanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = apiToOutgoingGatewayCtx(ctx, r)
	vars := mux.Vars(r)
	nodeIDStr := vars["node_id"]
	if nodeIDStr != "local" {
		nodeID, err := strconv.ParseInt(nodeIDStr, 10, 32)
		if err != nil || nodeID <= 0 {
			http.Error(w, "invalid node ID", http.StatusBadRequest)
			return
		}
	}

	ranges, ok := parseRangeIDs(r.URL.Query().Get("ranges"), w)
	if !ok {
		return
	}
	req := &serverpb.RangesRequest{
		NodeId:   nodeIDStr,
		RangeIDs: ranges,
	}
	limit, offset := getSimplePaginationValues(r)
	statusResp, next, err := a.status.rangesHelper(ctx, req, limit, offset)
	if err != nil {
		apiV2InternalError(ctx, err, w)
		return
	}
	resp := nodeRangesResponse{
		Ranges: make([]rangeInfo, 0, len(statusResp.Ranges)),
		Next:   next,
	}
	for _, r := range statusResp.Ranges {
		var ri rangeInfo
		ri.init(r)
		resp.Ranges = append(resp.Ranges, ri)
	}
	writeJSONResponse(ctx, w, 200, resp)
}

type responseError struct {
	ErrorMessage string         `json:"error_message"`
	NodeID       roachpb.NodeID `json:"node_id,omitempty"`
}

// Response struct for listHotRanges.
//
// swagger:model hotRangesResponse
type hotRangesResponse struct {
	RangesByNodeID map[string][]rangeDescriptorInfo `json:"ranges_by_node_id"`
	Errors         []responseError                  `json:"response_error,omitempty"`
	// Continuation token for the next paginated call. Use as the `start`
	// parameter.
	Next string `json:"next,omitempty"`
}

// swagger:operation GET /ranges/hot/ listHotRanges
//
// List hot ranges
//
// Lists information about hot ranges. If a list of range IDs
// is specified, only information about those ranges is returned.
//
// Client must be logged-in as a user with admin privileges.
//
// ---
// parameters:
// - name: node_id
//   in: query
//   type: integer
//   description: ID of node to query, or `local` for local node. If
//     unspecified, all nodes are queried.
//   required: false
// - name: limit
//   type: integer
//   in: query
//   description: Maximum number of results to return in this call.
//   required: false
// - name: start
//   type: string
//   in: query
//   description: Continuation token for results after a past limited run.
//   required: false
// produces:
// - application/json
// security:
// - api_session: []
// responses:
//   "200":
//     description: Hot ranges response.
//     schema:
//       "$ref": "#/definitions/hotRangesResponse"
func (a *apiV2Server) listHotRanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = apiToOutgoingGatewayCtx(ctx, r)
	nodeIDStr := r.URL.Query().Get("node_id")
	limit, start := getRPCPaginationValues(r)

	response := &hotRangesResponse{
		RangesByNodeID: make(map[string][]rangeDescriptorInfo),
	}
	var requestedNodes []roachpb.NodeID
	if len(nodeIDStr) > 0 {
		requestedNodeID, _, err := a.status.parseNodeID(nodeIDStr)
		if err != nil {
			http.Error(w, "invalid node ID", http.StatusBadRequest)
			return
		}
		requestedNodes = []roachpb.NodeID{requestedNodeID}
	}

	dialFn := func(ctx context.Context, nodeID roachpb.NodeID) (interface{}, error) {
		client, err := a.status.dialNode(ctx, nodeID)
		return client, err
	}
	remoteRequest := serverpb.HotRangesRequest{NodeID: "local"}
	nodeFn := func(ctx context.Context, client interface{}, nodeID roachpb.NodeID) (interface{}, error) {
		status := client.(serverpb.StatusClient)
		resp, err := status.HotRanges(ctx, &remoteRequest)
		if err != nil || resp == nil {
			return nil, err
		}
		rangeDescriptorInfos := make([]rangeDescriptorInfo, 0)
		for _, store := range resp.HotRangesByNodeID[nodeID].Stores {
			for _, hotRange := range store.HotRanges {
				var r rangeDescriptorInfo
				r.init(&hotRange.Desc)
				r.StoreID = store.StoreID
				r.QueriesPerSecond = hotRange.QueriesPerSecond
				rangeDescriptorInfos = append(rangeDescriptorInfos, r)
			}
		}
		sort.Slice(rangeDescriptorInfos, func(i, j int) bool {
			if rangeDescriptorInfos[i].StoreID == rangeDescriptorInfos[j].StoreID {
				return rangeDescriptorInfos[i].RangeID < rangeDescriptorInfos[j].RangeID
			}
			return rangeDescriptorInfos[i].StoreID < rangeDescriptorInfos[j].StoreID
		})
		return rangeDescriptorInfos, nil
	}
	responseFn := func(nodeID roachpb.NodeID, resp interface{}) {
		if hotRangesResp, ok := resp.([]rangeDescriptorInfo); ok {
			response.RangesByNodeID[nodeID.String()] = hotRangesResp
		}
	}
	errorFn := func(nodeID roachpb.NodeID, err error) {
		response.Errors = append(response.Errors, responseError{
			ErrorMessage: err.Error(),
			NodeID:       nodeID,
		})
	}

	next, err := a.status.paginatedIterateNodes(
		ctx, "hot ranges", limit, start, requestedNodes, dialFn,
		nodeFn, responseFn, errorFn)

	if err != nil {
		apiV2InternalError(ctx, err, w)
		return
	}
	var nextBytes []byte
	if nextBytes, err = next.MarshalText(); err != nil {
		response.Errors = append(response.Errors, responseError{ErrorMessage: err.Error()})
	} else {
		response.Next = string(nextBytes)
	}
	writeJSONResponse(ctx, w, 200, response)
}
