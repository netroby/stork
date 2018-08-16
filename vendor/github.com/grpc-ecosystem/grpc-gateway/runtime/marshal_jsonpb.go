package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// JSONPb is a Marshaler which marshals/unmarshals into/from JSON
// with the "github.com/golang/protobuf/jsonpb".
// It supports fully functionality of protobuf unlike JSONBuiltin.
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
//
// The NewDecoder method returns a DecoderWrapper, so the underlying
// *json.Decoder methods can be used.
=======
>>>>>>> Govendor update
type JSONPb jsonpb.Marshaler

// ContentType always returns "application/json".
func (*JSONPb) ContentType() string {
	return "application/json"
}

<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
// Marshal marshals "v" into JSON.
=======
// Marshal marshals "v" into JSON
// Currently it can marshal only proto.Message.
// TODO(yugui) Support fields of primitive types in a message.
>>>>>>> Govendor update
func (j *JSONPb) Marshal(v interface{}) ([]byte, error) {
	if _, ok := v.(proto.Message); !ok {
		return j.marshalNonProtoField(v)
	}

	var buf bytes.Buffer
	if err := j.marshalTo(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (j *JSONPb) marshalTo(w io.Writer, v interface{}) error {
	p, ok := v.(proto.Message)
	if !ok {
		buf, err := j.marshalNonProtoField(v)
		if err != nil {
			return err
		}
		_, err = w.Write(buf)
		return err
	}
	return (*jsonpb.Marshaler)(j).Marshal(w, p)
}

// marshalNonProto marshals a non-message field of a protobuf message.
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
// This function does not correctly marshals arbitrary data structure into JSON,
=======
// This function does not correctly marshals arbitary data structure into JSON,
>>>>>>> Govendor update
// but it is only capable of marshaling non-message field values of protobuf,
// i.e. primitive types, enums; pointers to primitives or enums; maps from
// integer/string types to primitives/enums/pointers to messages.
func (j *JSONPb) marshalNonProtoField(v interface{}) ([]byte, error) {
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	if v == nil {
		return []byte("null"), nil
	}
=======
>>>>>>> Govendor update
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []byte("null"), nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Map {
		m := make(map[string]*json.RawMessage)
		for _, k := range rv.MapKeys() {
			buf, err := j.Marshal(rv.MapIndex(k).Interface())
			if err != nil {
				return nil, err
			}
			m[fmt.Sprintf("%v", k.Interface())] = (*json.RawMessage)(&buf)
		}
		if j.Indent != "" {
			return json.MarshalIndent(m, "", j.Indent)
		}
		return json.Marshal(m)
	}
	if enum, ok := rv.Interface().(protoEnum); ok && !j.EnumsAsInts {
		return json.Marshal(enum.String())
	}
	return json.Marshal(rv.Interface())
}

// Unmarshal unmarshals JSON "data" into "v"
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
=======
// Currently it can marshal only proto.Message.
// TODO(yugui) Support fields of primitive types in a message.
>>>>>>> Govendor update
func (j *JSONPb) Unmarshal(data []byte, v interface{}) error {
	return unmarshalJSONPb(data, v)
}

// NewDecoder returns a Decoder which reads JSON stream from "r".
func (j *JSONPb) NewDecoder(r io.Reader) Decoder {
	d := json.NewDecoder(r)
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	return DecoderWrapper{Decoder: d}
}

// DecoderWrapper is a wrapper around a *json.Decoder that adds
// support for protos to the Decode method.
type DecoderWrapper struct {
	*json.Decoder
}

// Decode wraps the embedded decoder's Decode method to support
// protos using a jsonpb.Unmarshaler.
func (d DecoderWrapper) Decode(v interface{}) error {
	return decodeJSONPb(d.Decoder, v)
=======
	return DecoderFunc(func(v interface{}) error { return decodeJSONPb(d, v) })
>>>>>>> Govendor update
}

// NewEncoder returns an Encoder which writes JSON stream into "w".
func (j *JSONPb) NewEncoder(w io.Writer) Encoder {
	return EncoderFunc(func(v interface{}) error { return j.marshalTo(w, v) })
}

func unmarshalJSONPb(data []byte, v interface{}) error {
	d := json.NewDecoder(bytes.NewReader(data))
	return decodeJSONPb(d, v)
}

func decodeJSONPb(d *json.Decoder, v interface{}) error {
	p, ok := v.(proto.Message)
	if !ok {
		return decodeNonProtoField(d, v)
	}
	unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
	return unmarshaler.UnmarshalNext(d, p)
}

func decodeNonProtoField(d *json.Decoder, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%T is not a pointer", v)
	}
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		if rv.Type().ConvertibleTo(typeProtoMessage) {
			unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
			return unmarshaler.UnmarshalNext(d, rv.Interface().(proto.Message))
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Map {
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		conv, ok := convFromType[rv.Type().Key().Kind()]
		if !ok {
			return fmt.Errorf("unsupported type of map field key: %v", rv.Type().Key())
		}

		m := make(map[string]*json.RawMessage)
		if err := d.Decode(&m); err != nil {
			return err
		}
		for k, v := range m {
			result := conv.Call([]reflect.Value{reflect.ValueOf(k)})
			if err := result[1].Interface(); err != nil {
				return err.(error)
			}
			bk := result[0]
			bv := reflect.New(rv.Type().Elem())
			if err := unmarshalJSONPb([]byte(*v), bv.Interface()); err != nil {
				return err
			}
			rv.SetMapIndex(bk, bv.Elem())
		}
		return nil
	}
	if _, ok := rv.Interface().(protoEnum); ok {
		var repr interface{}
		if err := d.Decode(&repr); err != nil {
			return err
		}
		switch repr.(type) {
		case string:
			// TODO(yugui) Should use proto.StructProperties?
			return fmt.Errorf("unmarshaling of symbolic enum %q not supported: %T", repr, rv.Interface())
		case float64:
			rv.Set(reflect.ValueOf(int32(repr.(float64))).Convert(rv.Type()))
			return nil
		default:
			return fmt.Errorf("cannot assign %#v into Go type %T", repr, rv.Interface())
		}
	}
	return d.Decode(v)
}

type protoEnum interface {
	fmt.Stringer
	EnumDescriptor() ([]byte, []int)
}

var typeProtoMessage = reflect.TypeOf((*proto.Message)(nil)).Elem()
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64

// Delimiter for newline encoded JSON streams.
func (j *JSONPb) Delimiter() []byte {
	return []byte("\n")
}
=======
>>>>>>> Govendor update
