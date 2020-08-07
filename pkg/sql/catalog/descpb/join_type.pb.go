// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sql/catalog/descpb/join_type.proto

package descpb

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// JoinType is the particular type of a join (or join-like) operation. Not all
// values are used in all contexts.
type JoinType int32

const (
	JoinType_INNER       JoinType = 0
	JoinType_LEFT_OUTER  JoinType = 1
	JoinType_RIGHT_OUTER JoinType = 2
	JoinType_FULL_OUTER  JoinType = 3
	// A left semi join returns the rows from the left side that match at least
	// one row from the right side (as per equality columns and ON condition).
	JoinType_LEFT_SEMI JoinType = 4
	// A left anti join is an "inverted" semi join: it returns the rows from the
	// left side that don't match any columns on the right side (as per equality
	// columns and ON condition).
	JoinType_LEFT_ANTI JoinType = 5
	// INTERSECT_ALL is a special join-like operation that is only used for
	// INTERSECT ALL and INTERSECT operations.
	//
	// It is similar to a left semi join, except that if there are multiple left
	// rows that have the same values on the equality columns, only as many of
	// those are returned as there are matches on the right side.
	//
	// In practice, there is a one-to-one mapping between the left and right
	// columns (they are all equality columns).
	//
	// For example:
	//
	//       Left    Right    Result
	//       1       1        1
	//       1       2        2
	//       2       2        2
	//       2       3        3
	//       3       3
	//               3
	JoinType_INTERSECT_ALL JoinType = 6
	// EXCEPT_ALL is a special join-like operation that is only used for EXCEPT
	// ALL and EXCEPT operations.
	//
	// It is similar to a left anti join, except that if there are multiple left
	// rows that have the same values on the equality columns, only as many of
	// those are removed as there are matches on the right side.
	//
	// In practice, there is a one-to-one mapping between the left and right
	// columns (they are all equality columns).
	//
	// For example:
	//
	//       Left    Right    Result
	//       1       1        1
	//       1       2        2
	//       2       3        2
	//       2       3
	//       2       3
	//       3
	//       3
	//
	//
	// In practice, there is a one-to-one mapping between the left and right
	// columns (they are all equality columns).
	JoinType_EXCEPT_ALL JoinType = 7
)

var JoinType_name = map[int32]string{
	0: "INNER",
	1: "LEFT_OUTER",
	2: "RIGHT_OUTER",
	3: "FULL_OUTER",
	4: "LEFT_SEMI",
	5: "LEFT_ANTI",
	6: "INTERSECT_ALL",
	7: "EXCEPT_ALL",
}
var JoinType_value = map[string]int32{
	"INNER":         0,
	"LEFT_OUTER":    1,
	"RIGHT_OUTER":   2,
	"FULL_OUTER":    3,
	"LEFT_SEMI":     4,
	"LEFT_ANTI":     5,
	"INTERSECT_ALL": 6,
	"EXCEPT_ALL":    7,
}

func (x JoinType) Enum() *JoinType {
	p := new(JoinType)
	*p = x
	return p
}
func (x JoinType) String() string {
	return proto.EnumName(JoinType_name, int32(x))
}
func (x *JoinType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(JoinType_value, data, "JoinType")
	if err != nil {
		return err
	}
	*x = JoinType(value)
	return nil
}
func (JoinType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_join_type_9908800a37447c36, []int{0}
}

func init() {
	proto.RegisterEnum("cockroach.sql.sqlbase.JoinType", JoinType_name, JoinType_value)
}

func init() {
	proto.RegisterFile("sql/catalog/descpb/join_type.proto", fileDescriptor_join_type_9908800a37447c36)
}

var fileDescriptor_join_type_9908800a37447c36 = []byte{
	// 229 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x2a, 0x2e, 0xcc, 0xd1,
	0x4f, 0x4e, 0x2c, 0x49, 0xcc, 0xc9, 0x4f, 0xd7, 0x4f, 0x49, 0x2d, 0x4e, 0x2e, 0x48, 0xd2, 0xcf,
	0xca, 0xcf, 0xcc, 0x8b, 0x2f, 0xa9, 0x2c, 0x48, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12,
	0x4d, 0xce, 0x4f, 0xce, 0x2e, 0xca, 0x4f, 0x4c, 0xce, 0xd0, 0x2b, 0x2e, 0xcc, 0x01, 0xe1, 0xa4,
	0xc4, 0xe2, 0x54, 0xad, 0x76, 0x46, 0x2e, 0x0e, 0xaf, 0xfc, 0xcc, 0xbc, 0x90, 0xca, 0x82, 0x54,
	0x21, 0x4e, 0x2e, 0x56, 0x4f, 0x3f, 0x3f, 0xd7, 0x20, 0x01, 0x06, 0x21, 0x3e, 0x2e, 0x2e, 0x1f,
	0x57, 0xb7, 0x90, 0x78, 0xff, 0xd0, 0x10, 0xd7, 0x20, 0x01, 0x46, 0x21, 0x7e, 0x2e, 0xee, 0x20,
	0x4f, 0x77, 0x0f, 0x98, 0x00, 0x13, 0x48, 0x81, 0x5b, 0xa8, 0x8f, 0x0f, 0x94, 0xcf, 0x2c, 0xc4,
	0xcb, 0xc5, 0x09, 0xd6, 0x10, 0xec, 0xea, 0xeb, 0x29, 0xc0, 0x02, 0xe7, 0x3a, 0xfa, 0x85, 0x78,
	0x0a, 0xb0, 0x0a, 0x09, 0x72, 0xf1, 0x7a, 0xfa, 0x85, 0xb8, 0x06, 0x05, 0xbb, 0x3a, 0x87, 0xc4,
	0x3b, 0xfa, 0xf8, 0x08, 0xb0, 0x81, 0x0c, 0x70, 0x8d, 0x70, 0x76, 0x0d, 0x80, 0xf0, 0xd9, 0x9d,
	0x34, 0x4e, 0x3c, 0x94, 0x63, 0x38, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x1b, 0x8f,
	0xe4, 0x18, 0x1f, 0x3c, 0x92, 0x63, 0x9c, 0xf0, 0x58, 0x8e, 0xe1, 0xc2, 0x63, 0x39, 0x86, 0x1b,
	0x8f, 0xe5, 0x18, 0xa2, 0xd8, 0x20, 0x5e, 0x03, 0x04, 0x00, 0x00, 0xff, 0xff, 0x5e, 0x1b, 0x54,
	0xba, 0xef, 0x00, 0x00, 0x00,
}