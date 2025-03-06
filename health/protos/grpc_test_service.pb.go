// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.3
// source: grpc_test_service.proto

package protos

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// FooRequest params needed to make request
type FooRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bar           string                 `protobuf:"bytes,1,opt,name=bar,proto3" json:"bar,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FooRequest) Reset() {
	*x = FooRequest{}
	mi := &file_grpc_test_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FooRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FooRequest) ProtoMessage() {}

func (x *FooRequest) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_test_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FooRequest.ProtoReflect.Descriptor instead.
func (*FooRequest) Descriptor() ([]byte, []int) {
	return file_grpc_test_service_proto_rawDescGZIP(), []int{0}
}

func (x *FooRequest) GetBar() string {
	if x != nil {
		return x.Bar
	}
	return ""
}

// FooResponse result of the request
type FooResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FooResponse) Reset() {
	*x = FooResponse{}
	mi := &file_grpc_test_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FooResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FooResponse) ProtoMessage() {}

func (x *FooResponse) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_test_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FooResponse.ProtoReflect.Descriptor instead.
func (*FooResponse) Descriptor() ([]byte, []int) {
	return file_grpc_test_service_proto_rawDescGZIP(), []int{1}
}

var File_grpc_test_service_proto protoreflect.FileDescriptor

var file_grpc_test_service_proto_rawDesc = string([]byte{
	0x0a, 0x17, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x22, 0x1e, 0x0a, 0x0a, 0x46, 0x6f, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x10, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x62, 0x61,
	0x72, 0x22, 0x0d, 0x0a, 0x0b, 0x46, 0x6f, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x32, 0x3d, 0x0a, 0x0b, 0x54, 0x65, 0x73, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x2e, 0x0a, 0x03, 0x46, 0x6f, 0x6f, 0x12, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e,
	0x46, 0x6f, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x13, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x73, 0x2e, 0x46, 0x6f, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42,
	0x0b, 0x5a, 0x09, 0x2e, 0x2f, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_grpc_test_service_proto_rawDescOnce sync.Once
	file_grpc_test_service_proto_rawDescData []byte
)

func file_grpc_test_service_proto_rawDescGZIP() []byte {
	file_grpc_test_service_proto_rawDescOnce.Do(func() {
		file_grpc_test_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_grpc_test_service_proto_rawDesc), len(file_grpc_test_service_proto_rawDesc)))
	})
	return file_grpc_test_service_proto_rawDescData
}

var file_grpc_test_service_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_grpc_test_service_proto_goTypes = []any{
	(*FooRequest)(nil),  // 0: protos.FooRequest
	(*FooResponse)(nil), // 1: protos.FooResponse
}
var file_grpc_test_service_proto_depIdxs = []int32{
	0, // 0: protos.TestService.Foo:input_type -> protos.FooRequest
	1, // 1: protos.TestService.Foo:output_type -> protos.FooResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_grpc_test_service_proto_init() }
func file_grpc_test_service_proto_init() {
	if File_grpc_test_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_grpc_test_service_proto_rawDesc), len(file_grpc_test_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_grpc_test_service_proto_goTypes,
		DependencyIndexes: file_grpc_test_service_proto_depIdxs,
		MessageInfos:      file_grpc_test_service_proto_msgTypes,
	}.Build()
	File_grpc_test_service_proto = out.File
	file_grpc_test_service_proto_goTypes = nil
	file_grpc_test_service_proto_depIdxs = nil
}
