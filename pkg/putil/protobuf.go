package putil

import (
	reflect "reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

var (
	marsharler   *protojson.MarshalOptions
	unmarsharler *protojson.UnmarshalOptions
)

func getMarsharler() *protojson.MarshalOptions {
	if marsharler == nil {
		marsharler = &protojson.MarshalOptions{EmitUnpopulated: true, UseEnumNumbers: true}
	}
	return marsharler
}
func getUnmarsharler() *protojson.UnmarshalOptions {
	if unmarsharler == nil {
		unmarsharler = &protojson.UnmarshalOptions{DiscardUnknown: true}
	}
	return unmarsharler
}

const marsharlToJson = true

// --------------------------------------------------
func StringToMessage(str *string, msg proto.Message) error {
	if marsharlToJson {
		return getUnmarsharler().Unmarshal([]byte(*str), msg)
	} else {
		return proto.Unmarshal([]byte(*str), msg)
	}
}

func MessageToString(msg proto.Message, str *string) error {
	var err error
	var tmpByte []byte
	if marsharlToJson {
		tmpByte, err = getMarsharler().Marshal(msg)
	} else {
		tmpByte, err = proto.Marshal(msg)
	}
	if err != nil {
		return err
	}
	*str = string(tmpByte)
	return nil
}

func MessageToCleanString(msg proto.Message, str *string) error {
	var err error
	var tmpByte []byte
	if marsharlToJson {
		marsharler = &protojson.MarshalOptions{EmitUnpopulated: false, UseEnumNumbers: true}
		tmpByte, err = marsharler.Marshal(msg)
	} else {
		tmpByte, err = proto.Marshal(msg)
	}
	if err != nil {
		return err
	}
	*str = string(tmpByte)
	return nil
}

// --------------------------------------------------
func IsFuncParamPB(f any) bool {
	reqType := reflect.ValueOf(f).Type().In(1)
	reqVal := reflect.New(reqType) //point to pb.xxxRequest
	return IsPB(reqVal.Type())
}

func IsPB(t reflect.Type) bool {
	protoMessageType := reflect.TypeOf((*protoreflect.ProtoMessage)(nil)).Elem()
	// fmt.Println(protoMessageType)
	return t.Implements(protoMessageType)
}
