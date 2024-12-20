// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        (unknown)
// source: internal/deadletter.proto

package internalpb

import (
	goaktpb "github.com/tochemey/goakt/v2/goaktpb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetDeadletters struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetDeadletters) Reset() {
	*x = GetDeadletters{}
	mi := &file_internal_deadletter_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetDeadletters) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetDeadletters) ProtoMessage() {}

func (x *GetDeadletters) ProtoReflect() protoreflect.Message {
	mi := &file_internal_deadletter_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetDeadletters.ProtoReflect.Descriptor instead.
func (*GetDeadletters) Descriptor() ([]byte, []int) {
	return file_internal_deadletter_proto_rawDescGZIP(), []int{0}
}

type EmitDeadletter struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Specifies the deadletter to emit
	Deadletter    *goaktpb.Deadletter `protobuf:"bytes,1,opt,name=deadletter,proto3" json:"deadletter,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EmitDeadletter) Reset() {
	*x = EmitDeadletter{}
	mi := &file_internal_deadletter_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EmitDeadletter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmitDeadletter) ProtoMessage() {}

func (x *EmitDeadletter) ProtoReflect() protoreflect.Message {
	mi := &file_internal_deadletter_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmitDeadletter.ProtoReflect.Descriptor instead.
func (*EmitDeadletter) Descriptor() ([]byte, []int) {
	return file_internal_deadletter_proto_rawDescGZIP(), []int{1}
}

func (x *EmitDeadletter) GetDeadletter() *goaktpb.Deadletter {
	if x != nil {
		return x.Deadletter
	}
	return nil
}

type GetDeadlettersCount struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// optional for a specific actor
	ActorId       *string `protobuf:"bytes,1,opt,name=actor_id,json=actorId,proto3,oneof" json:"actor_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetDeadlettersCount) Reset() {
	*x = GetDeadlettersCount{}
	mi := &file_internal_deadletter_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetDeadlettersCount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetDeadlettersCount) ProtoMessage() {}

func (x *GetDeadlettersCount) ProtoReflect() protoreflect.Message {
	mi := &file_internal_deadletter_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetDeadlettersCount.ProtoReflect.Descriptor instead.
func (*GetDeadlettersCount) Descriptor() ([]byte, []int) {
	return file_internal_deadletter_proto_rawDescGZIP(), []int{2}
}

func (x *GetDeadlettersCount) GetActorId() string {
	if x != nil && x.ActorId != nil {
		return *x.ActorId
	}
	return ""
}

type DeadlettersCount struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Specifies the total count
	TotalCount    int64 `protobuf:"varint,1,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeadlettersCount) Reset() {
	*x = DeadlettersCount{}
	mi := &file_internal_deadletter_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeadlettersCount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeadlettersCount) ProtoMessage() {}

func (x *DeadlettersCount) ProtoReflect() protoreflect.Message {
	mi := &file_internal_deadletter_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeadlettersCount.ProtoReflect.Descriptor instead.
func (*DeadlettersCount) Descriptor() ([]byte, []int) {
	return file_internal_deadletter_proto_rawDescGZIP(), []int{3}
}

func (x *DeadlettersCount) GetTotalCount() int64 {
	if x != nil {
		return x.TotalCount
	}
	return 0
}

var File_internal_deadletter_proto protoreflect.FileDescriptor

var file_internal_deadletter_proto_rawDesc = []byte{
	0x0a, 0x19, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x64, 0x65, 0x61, 0x64, 0x6c,
	0x65, 0x74, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x1a, 0x11, 0x67, 0x6f, 0x61, 0x6b, 0x74, 0x2f, 0x67,
	0x6f, 0x61, 0x6b, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x10, 0x0a, 0x0e, 0x47, 0x65,
	0x74, 0x44, 0x65, 0x61, 0x64, 0x6c, 0x65, 0x74, 0x74, 0x65, 0x72, 0x73, 0x22, 0x45, 0x0a, 0x0e,
	0x45, 0x6d, 0x69, 0x74, 0x44, 0x65, 0x61, 0x64, 0x6c, 0x65, 0x74, 0x74, 0x65, 0x72, 0x12, 0x33,
	0x0a, 0x0a, 0x64, 0x65, 0x61, 0x64, 0x6c, 0x65, 0x74, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x13, 0x2e, 0x67, 0x6f, 0x61, 0x6b, 0x74, 0x70, 0x62, 0x2e, 0x44, 0x65, 0x61,
	0x64, 0x6c, 0x65, 0x74, 0x74, 0x65, 0x72, 0x52, 0x0a, 0x64, 0x65, 0x61, 0x64, 0x6c, 0x65, 0x74,
	0x74, 0x65, 0x72, 0x22, 0x42, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x44, 0x65, 0x61, 0x64, 0x6c, 0x65,
	0x74, 0x74, 0x65, 0x72, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1e, 0x0a, 0x08, 0x61, 0x63,
	0x74, 0x6f, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07,
	0x61, 0x63, 0x74, 0x6f, 0x72, 0x49, 0x64, 0x88, 0x01, 0x01, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x61,
	0x63, 0x74, 0x6f, 0x72, 0x5f, 0x69, 0x64, 0x22, 0x33, 0x0a, 0x10, 0x44, 0x65, 0x61, 0x64, 0x6c,
	0x65, 0x74, 0x74, 0x65, 0x72, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x74,
	0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x0a, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x42, 0xa8, 0x01, 0x0a,
	0x0e, 0x63, 0x6f, 0x6d, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x42,
	0x0f, 0x44, 0x65, 0x61, 0x64, 0x6c, 0x65, 0x74, 0x74, 0x65, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x48, 0x02, 0x50, 0x01, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x74, 0x6f, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x79, 0x2f, 0x67, 0x6f, 0x61, 0x6b, 0x74, 0x2f,
	0x76, 0x32, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x3b, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70,
	0x62, 0xa2, 0x02, 0x03, 0x49, 0x58, 0x58, 0xaa, 0x02, 0x0a, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x70, 0x62, 0xca, 0x02, 0x0a, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70,
	0x62, 0xe2, 0x02, 0x16, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_deadletter_proto_rawDescOnce sync.Once
	file_internal_deadletter_proto_rawDescData = file_internal_deadletter_proto_rawDesc
)

func file_internal_deadletter_proto_rawDescGZIP() []byte {
	file_internal_deadletter_proto_rawDescOnce.Do(func() {
		file_internal_deadletter_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_deadletter_proto_rawDescData)
	})
	return file_internal_deadletter_proto_rawDescData
}

var file_internal_deadletter_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_internal_deadletter_proto_goTypes = []any{
	(*GetDeadletters)(nil),      // 0: internalpb.GetDeadletters
	(*EmitDeadletter)(nil),      // 1: internalpb.EmitDeadletter
	(*GetDeadlettersCount)(nil), // 2: internalpb.GetDeadlettersCount
	(*DeadlettersCount)(nil),    // 3: internalpb.DeadlettersCount
	(*goaktpb.Deadletter)(nil),  // 4: goaktpb.Deadletter
}
var file_internal_deadletter_proto_depIdxs = []int32{
	4, // 0: internalpb.EmitDeadletter.deadletter:type_name -> goaktpb.Deadletter
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_internal_deadletter_proto_init() }
func file_internal_deadletter_proto_init() {
	if File_internal_deadletter_proto != nil {
		return
	}
	file_internal_deadletter_proto_msgTypes[2].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_deadletter_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_deadletter_proto_goTypes,
		DependencyIndexes: file_internal_deadletter_proto_depIdxs,
		MessageInfos:      file_internal_deadletter_proto_msgTypes,
	}.Build()
	File_internal_deadletter_proto = out.File
	file_internal_deadletter_proto_rawDesc = nil
	file_internal_deadletter_proto_goTypes = nil
	file_internal_deadletter_proto_depIdxs = nil
}