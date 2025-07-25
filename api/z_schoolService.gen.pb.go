// Code generated by tools/genCURD. DO NOT EDIT.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.0
// source: z_schoolService.gen.proto

package api

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

// --------------------------------------------------
// tbl : course_swap_request
type CourseSwapRequestInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ID            int32                  `protobuf:"varint,1,opt,name=iD,proto3" json:"iD,omitempty"`
	SrcTeacher    string                 `protobuf:"bytes,2,opt,name=srcTeacher,proto3" json:"srcTeacher,omitempty"`
	SrcDate       string                 `protobuf:"bytes,3,opt,name=srcDate,proto3" json:"srcDate,omitempty"`
	SrcCourseNum  int32                  `protobuf:"varint,4,opt,name=srcCourseNum,proto3" json:"srcCourseNum,omitempty"`
	SrcCourse     string                 `protobuf:"bytes,5,opt,name=srcCourse,proto3" json:"srcCourse,omitempty"`
	SrcClass      string                 `protobuf:"bytes,6,opt,name=srcClass,proto3" json:"srcClass,omitempty"`
	DstTeacher    string                 `protobuf:"bytes,7,opt,name=dstTeacher,proto3" json:"dstTeacher,omitempty"`
	DstDate       string                 `protobuf:"bytes,8,opt,name=dstDate,proto3" json:"dstDate,omitempty"`
	DstCourseNum  int32                  `protobuf:"varint,9,opt,name=dstCourseNum,proto3" json:"dstCourseNum,omitempty"`
	DstCourse     string                 `protobuf:"bytes,10,opt,name=dstCourse,proto3" json:"dstCourse,omitempty"`
	DstClass      string                 `protobuf:"bytes,11,opt,name=dstClass,proto3" json:"dstClass,omitempty"`
	CreateTime    string                 `protobuf:"bytes,12,opt,name=createTime,proto3" json:"createTime,omitempty"`
	Status        int32                  `protobuf:"varint,13,opt,name=status,proto3" json:"status,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CourseSwapRequestInfo) Reset() {
	*x = CourseSwapRequestInfo{}
	mi := &file_z_schoolService_gen_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CourseSwapRequestInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CourseSwapRequestInfo) ProtoMessage() {}

func (x *CourseSwapRequestInfo) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CourseSwapRequestInfo.ProtoReflect.Descriptor instead.
func (*CourseSwapRequestInfo) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{0}
}

func (x *CourseSwapRequestInfo) GetID() int32 {
	if x != nil {
		return x.ID
	}
	return 0
}

func (x *CourseSwapRequestInfo) GetSrcTeacher() string {
	if x != nil {
		return x.SrcTeacher
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetSrcDate() string {
	if x != nil {
		return x.SrcDate
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetSrcCourseNum() int32 {
	if x != nil {
		return x.SrcCourseNum
	}
	return 0
}

func (x *CourseSwapRequestInfo) GetSrcCourse() string {
	if x != nil {
		return x.SrcCourse
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetSrcClass() string {
	if x != nil {
		return x.SrcClass
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetDstTeacher() string {
	if x != nil {
		return x.DstTeacher
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetDstDate() string {
	if x != nil {
		return x.DstDate
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetDstCourseNum() int32 {
	if x != nil {
		return x.DstCourseNum
	}
	return 0
}

func (x *CourseSwapRequestInfo) GetDstCourse() string {
	if x != nil {
		return x.DstCourse
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetDstClass() string {
	if x != nil {
		return x.DstClass
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetCreateTime() string {
	if x != nil {
		return x.CreateTime
	}
	return ""
}

func (x *CourseSwapRequestInfo) GetStatus() int32 {
	if x != nil {
		return x.Status
	}
	return 0
}

type AddCourseSwapRequestRequest struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	CourseSwapRequest *CourseSwapRequestInfo `protobuf:"bytes,1,opt,name=courseSwapRequest,proto3" json:"courseSwapRequest,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *AddCourseSwapRequestRequest) Reset() {
	*x = AddCourseSwapRequestRequest{}
	mi := &file_z_schoolService_gen_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddCourseSwapRequestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCourseSwapRequestRequest) ProtoMessage() {}

func (x *AddCourseSwapRequestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCourseSwapRequestRequest.ProtoReflect.Descriptor instead.
func (*AddCourseSwapRequestRequest) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{1}
}

func (x *AddCourseSwapRequestRequest) GetCourseSwapRequest() *CourseSwapRequestInfo {
	if x != nil {
		return x.CourseSwapRequest
	}
	return nil
}

type AddCourseSwapRequestResponse struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	CourseSwapRequest *CourseSwapRequestInfo `protobuf:"bytes,1,opt,name=courseSwapRequest,proto3" json:"courseSwapRequest,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *AddCourseSwapRequestResponse) Reset() {
	*x = AddCourseSwapRequestResponse{}
	mi := &file_z_schoolService_gen_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddCourseSwapRequestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCourseSwapRequestResponse) ProtoMessage() {}

func (x *AddCourseSwapRequestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCourseSwapRequestResponse.ProtoReflect.Descriptor instead.
func (*AddCourseSwapRequestResponse) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{2}
}

func (x *AddCourseSwapRequestResponse) GetCourseSwapRequest() *CourseSwapRequestInfo {
	if x != nil {
		return x.CourseSwapRequest
	}
	return nil
}

type GetCourseSwapRequestListRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	IDList        []int32                `protobuf:"varint,1,rep,packed,name=iDList,proto3" json:"iDList,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCourseSwapRequestListRequest) Reset() {
	*x = GetCourseSwapRequestListRequest{}
	mi := &file_z_schoolService_gen_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCourseSwapRequestListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCourseSwapRequestListRequest) ProtoMessage() {}

func (x *GetCourseSwapRequestListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCourseSwapRequestListRequest.ProtoReflect.Descriptor instead.
func (*GetCourseSwapRequestListRequest) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{3}
}

func (x *GetCourseSwapRequestListRequest) GetIDList() []int32 {
	if x != nil {
		return x.IDList
	}
	return nil
}

type GetCourseSwapRequestListResponse struct {
	state                 protoimpl.MessageState   `protogen:"open.v1"`
	CourseSwapRequestList []*CourseSwapRequestInfo `protobuf:"bytes,1,rep,name=courseSwapRequestList,proto3" json:"courseSwapRequestList,omitempty"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *GetCourseSwapRequestListResponse) Reset() {
	*x = GetCourseSwapRequestListResponse{}
	mi := &file_z_schoolService_gen_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCourseSwapRequestListResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCourseSwapRequestListResponse) ProtoMessage() {}

func (x *GetCourseSwapRequestListResponse) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCourseSwapRequestListResponse.ProtoReflect.Descriptor instead.
func (*GetCourseSwapRequestListResponse) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{4}
}

func (x *GetCourseSwapRequestListResponse) GetCourseSwapRequestList() []*CourseSwapRequestInfo {
	if x != nil {
		return x.CourseSwapRequestList
	}
	return nil
}

type UpdateCourseSwapRequestRequest struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	CourseSwapRequest *CourseSwapRequestInfo `protobuf:"bytes,1,opt,name=courseSwapRequest,proto3" json:"courseSwapRequest,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *UpdateCourseSwapRequestRequest) Reset() {
	*x = UpdateCourseSwapRequestRequest{}
	mi := &file_z_schoolService_gen_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateCourseSwapRequestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateCourseSwapRequestRequest) ProtoMessage() {}

func (x *UpdateCourseSwapRequestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateCourseSwapRequestRequest.ProtoReflect.Descriptor instead.
func (*UpdateCourseSwapRequestRequest) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{5}
}

func (x *UpdateCourseSwapRequestRequest) GetCourseSwapRequest() *CourseSwapRequestInfo {
	if x != nil {
		return x.CourseSwapRequest
	}
	return nil
}

type UpdateCourseSwapRequestResponse struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	CourseSwapRequest *CourseSwapRequestInfo `protobuf:"bytes,1,opt,name=courseSwapRequest,proto3" json:"courseSwapRequest,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *UpdateCourseSwapRequestResponse) Reset() {
	*x = UpdateCourseSwapRequestResponse{}
	mi := &file_z_schoolService_gen_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateCourseSwapRequestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateCourseSwapRequestResponse) ProtoMessage() {}

func (x *UpdateCourseSwapRequestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateCourseSwapRequestResponse.ProtoReflect.Descriptor instead.
func (*UpdateCourseSwapRequestResponse) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{6}
}

func (x *UpdateCourseSwapRequestResponse) GetCourseSwapRequest() *CourseSwapRequestInfo {
	if x != nil {
		return x.CourseSwapRequest
	}
	return nil
}

type DelCourseSwapRequestByIDListRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	IDList        []int32                `protobuf:"varint,1,rep,packed,name=iDList,proto3" json:"iDList,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DelCourseSwapRequestByIDListRequest) Reset() {
	*x = DelCourseSwapRequestByIDListRequest{}
	mi := &file_z_schoolService_gen_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DelCourseSwapRequestByIDListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DelCourseSwapRequestByIDListRequest) ProtoMessage() {}

func (x *DelCourseSwapRequestByIDListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_z_schoolService_gen_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DelCourseSwapRequestByIDListRequest.ProtoReflect.Descriptor instead.
func (*DelCourseSwapRequestByIDListRequest) Descriptor() ([]byte, []int) {
	return file_z_schoolService_gen_proto_rawDescGZIP(), []int{7}
}

func (x *DelCourseSwapRequestByIDListRequest) GetIDList() []int32 {
	if x != nil {
		return x.IDList
	}
	return nil
}

var File_z_schoolService_gen_proto protoreflect.FileDescriptor

const file_z_schoolService_gen_proto_rawDesc = "" +
	"\n" +
	"\x19z_schoolService.gen.proto\x12\x03api\x1a\x1cgoogle/api/annotations.proto\x1a\fcommon.proto\"\x8f\x03\n" +
	"\x15CourseSwapRequestInfo\x12\x0e\n" +
	"\x02iD\x18\x01 \x01(\x05R\x02iD\x12\x1e\n" +
	"\n" +
	"srcTeacher\x18\x02 \x01(\tR\n" +
	"srcTeacher\x12\x18\n" +
	"\asrcDate\x18\x03 \x01(\tR\asrcDate\x12\"\n" +
	"\fsrcCourseNum\x18\x04 \x01(\x05R\fsrcCourseNum\x12\x1c\n" +
	"\tsrcCourse\x18\x05 \x01(\tR\tsrcCourse\x12\x1a\n" +
	"\bsrcClass\x18\x06 \x01(\tR\bsrcClass\x12\x1e\n" +
	"\n" +
	"dstTeacher\x18\a \x01(\tR\n" +
	"dstTeacher\x12\x18\n" +
	"\adstDate\x18\b \x01(\tR\adstDate\x12\"\n" +
	"\fdstCourseNum\x18\t \x01(\x05R\fdstCourseNum\x12\x1c\n" +
	"\tdstCourse\x18\n" +
	" \x01(\tR\tdstCourse\x12\x1a\n" +
	"\bdstClass\x18\v \x01(\tR\bdstClass\x12\x1e\n" +
	"\n" +
	"createTime\x18\f \x01(\tR\n" +
	"createTime\x12\x16\n" +
	"\x06status\x18\r \x01(\x05R\x06status\"g\n" +
	"\x1bAddCourseSwapRequestRequest\x12H\n" +
	"\x11courseSwapRequest\x18\x01 \x01(\v2\x1a.api.CourseSwapRequestInfoR\x11courseSwapRequest\"h\n" +
	"\x1cAddCourseSwapRequestResponse\x12H\n" +
	"\x11courseSwapRequest\x18\x01 \x01(\v2\x1a.api.CourseSwapRequestInfoR\x11courseSwapRequest\"9\n" +
	"\x1fGetCourseSwapRequestListRequest\x12\x16\n" +
	"\x06iDList\x18\x01 \x03(\x05R\x06iDList\"t\n" +
	" GetCourseSwapRequestListResponse\x12P\n" +
	"\x15courseSwapRequestList\x18\x01 \x03(\v2\x1a.api.CourseSwapRequestInfoR\x15courseSwapRequestList\"j\n" +
	"\x1eUpdateCourseSwapRequestRequest\x12H\n" +
	"\x11courseSwapRequest\x18\x01 \x01(\v2\x1a.api.CourseSwapRequestInfoR\x11courseSwapRequest\"k\n" +
	"\x1fUpdateCourseSwapRequestResponse\x12H\n" +
	"\x11courseSwapRequest\x18\x01 \x01(\v2\x1a.api.CourseSwapRequestInfoR\x11courseSwapRequest\"=\n" +
	"#DelCourseSwapRequestByIDListRequest\x12\x16\n" +
	"\x06iDList\x18\x01 \x03(\x05R\x06iDList2\x8e\x04\n" +
	"\n" +
	"schoolCURD\x12|\n" +
	"\x14AddCourseSwapRequest\x12 .api.AddCourseSwapRequestRequest\x1a!.api.AddCourseSwapRequestResponse\"\x1f\x82\xd3\xe4\x93\x02\x19:\x01*\"\x14/course-swap-request\x12\x85\x01\n" +
	"\x18GetCourseSwapRequestList\x12$.api.GetCourseSwapRequestListRequest\x1a%.api.GetCourseSwapRequestListResponse\"\x1c\x82\xd3\xe4\x93\x02\x16\x12\x14/course-swap-request\x12\x85\x01\n" +
	"\x17UpdateCourseSwapRequest\x12#.api.UpdateCourseSwapRequestRequest\x1a$.api.UpdateCourseSwapRequestResponse\"\x1f\x82\xd3\xe4\x93\x02\x19:\x01*2\x14/course-swap-request\x12r\n" +
	"\x1cDelCourseSwapRequestByIDList\x12(.api.DelCourseSwapRequestByIDListRequest\x1a\n" +
	".api.Empty\"\x1c\x82\xd3\xe4\x93\x02\x16*\x14/course-swap-requestB\rZ\vpgo/api;apib\x06proto3"

var (
	file_z_schoolService_gen_proto_rawDescOnce sync.Once
	file_z_schoolService_gen_proto_rawDescData []byte
)

func file_z_schoolService_gen_proto_rawDescGZIP() []byte {
	file_z_schoolService_gen_proto_rawDescOnce.Do(func() {
		file_z_schoolService_gen_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_z_schoolService_gen_proto_rawDesc), len(file_z_schoolService_gen_proto_rawDesc)))
	})
	return file_z_schoolService_gen_proto_rawDescData
}

var file_z_schoolService_gen_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_z_schoolService_gen_proto_goTypes = []any{
	(*CourseSwapRequestInfo)(nil),               // 0: api.CourseSwapRequestInfo
	(*AddCourseSwapRequestRequest)(nil),         // 1: api.AddCourseSwapRequestRequest
	(*AddCourseSwapRequestResponse)(nil),        // 2: api.AddCourseSwapRequestResponse
	(*GetCourseSwapRequestListRequest)(nil),     // 3: api.GetCourseSwapRequestListRequest
	(*GetCourseSwapRequestListResponse)(nil),    // 4: api.GetCourseSwapRequestListResponse
	(*UpdateCourseSwapRequestRequest)(nil),      // 5: api.UpdateCourseSwapRequestRequest
	(*UpdateCourseSwapRequestResponse)(nil),     // 6: api.UpdateCourseSwapRequestResponse
	(*DelCourseSwapRequestByIDListRequest)(nil), // 7: api.DelCourseSwapRequestByIDListRequest
	(*Empty)(nil), // 8: api.Empty
}
var file_z_schoolService_gen_proto_depIdxs = []int32{
	0, // 0: api.AddCourseSwapRequestRequest.courseSwapRequest:type_name -> api.CourseSwapRequestInfo
	0, // 1: api.AddCourseSwapRequestResponse.courseSwapRequest:type_name -> api.CourseSwapRequestInfo
	0, // 2: api.GetCourseSwapRequestListResponse.courseSwapRequestList:type_name -> api.CourseSwapRequestInfo
	0, // 3: api.UpdateCourseSwapRequestRequest.courseSwapRequest:type_name -> api.CourseSwapRequestInfo
	0, // 4: api.UpdateCourseSwapRequestResponse.courseSwapRequest:type_name -> api.CourseSwapRequestInfo
	1, // 5: api.schoolCURD.AddCourseSwapRequest:input_type -> api.AddCourseSwapRequestRequest
	3, // 6: api.schoolCURD.GetCourseSwapRequestList:input_type -> api.GetCourseSwapRequestListRequest
	5, // 7: api.schoolCURD.UpdateCourseSwapRequest:input_type -> api.UpdateCourseSwapRequestRequest
	7, // 8: api.schoolCURD.DelCourseSwapRequestByIDList:input_type -> api.DelCourseSwapRequestByIDListRequest
	2, // 9: api.schoolCURD.AddCourseSwapRequest:output_type -> api.AddCourseSwapRequestResponse
	4, // 10: api.schoolCURD.GetCourseSwapRequestList:output_type -> api.GetCourseSwapRequestListResponse
	6, // 11: api.schoolCURD.UpdateCourseSwapRequest:output_type -> api.UpdateCourseSwapRequestResponse
	8, // 12: api.schoolCURD.DelCourseSwapRequestByIDList:output_type -> api.Empty
	9, // [9:13] is the sub-list for method output_type
	5, // [5:9] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_z_schoolService_gen_proto_init() }
func file_z_schoolService_gen_proto_init() {
	if File_z_schoolService_gen_proto != nil {
		return
	}
	file_common_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_z_schoolService_gen_proto_rawDesc), len(file_z_schoolService_gen_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_z_schoolService_gen_proto_goTypes,
		DependencyIndexes: file_z_schoolService_gen_proto_depIdxs,
		MessageInfos:      file_z_schoolService_gen_proto_msgTypes,
	}.Build()
	File_z_schoolService_gen_proto = out.File
	file_z_schoolService_gen_proto_goTypes = nil
	file_z_schoolService_gen_proto_depIdxs = nil
}
