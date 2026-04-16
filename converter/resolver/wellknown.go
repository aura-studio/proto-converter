package resolver

// WellKnownTypes is the mapping from Google well-known proto types to their import paths.
var WellKnownTypes = map[string]string{
	"google.protobuf.Timestamp":   "google/protobuf/timestamp.proto",
	"google.protobuf.Duration":    "google/protobuf/duration.proto",
	"google.protobuf.Any":         "google/protobuf/any.proto",
	"google.protobuf.Empty":       "google/protobuf/empty.proto",
	"google.protobuf.Struct":      "google/protobuf/struct.proto",
	"google.protobuf.Value":       "google/protobuf/struct.proto",
	"google.protobuf.ListValue":   "google/protobuf/struct.proto",
	"google.protobuf.Int32Value":  "google/protobuf/wrappers.proto",
	"google.protobuf.Int64Value":  "google/protobuf/wrappers.proto",
	"google.protobuf.StringValue": "google/protobuf/wrappers.proto",
	"google.protobuf.BoolValue":   "google/protobuf/wrappers.proto",
	"google.protobuf.BytesValue":  "google/protobuf/wrappers.proto",
	"google.protobuf.UInt32Value": "google/protobuf/wrappers.proto",
	"google.protobuf.UInt64Value": "google/protobuf/wrappers.proto",
	"google.protobuf.FloatValue":  "google/protobuf/wrappers.proto",
	"google.protobuf.DoubleValue": "google/protobuf/wrappers.proto",
}

// ScalarTypes is the set of proto scalar types.
var ScalarTypes = map[string]struct{}{
	"double": {}, "float": {}, "int32": {}, "int64": {},
	"uint32": {}, "uint64": {}, "sint32": {}, "sint64": {},
	"fixed32": {}, "fixed64": {}, "sfixed32": {}, "sfixed64": {},
	"bool": {}, "string": {}, "bytes": {},
}
