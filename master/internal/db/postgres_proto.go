package db

import (
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (db *PgDB) QueryProto(queryName string, v interface{}, args ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error {
		input := make(map[string]interface{})

		if err := rows.MapScan(input); err != nil {
			return err
		}
		val = reflect.ValueOf(val).Elem().Interface()
		parsed, err := convert(val.(proto.Message).ProtoReflect().Descriptor(), input)
		if err != nil {
			return err
		}
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(parsed.Interface()))
		return nil
	}
	return db.queryRows(queryName, parser, v, args...)
}

func convert(desc pref.Descriptor, input interface{}) (pref.Value, error) {
	switch d := desc.(type) {
	case pref.EnumDescriptor:
		return convertEnum(d, input)
	case pref.MessageDescriptor:
		return convertStruct(d, input)
	case pref.FieldDescriptor:
		return convertField(d, input)
	default:
		return pref.Value{}, errors.Errorf("unknown descriptor: %s", desc.FullName())
	}
}

func convertEnum(d pref.EnumDescriptor, input interface{}) (pref.Value, error) {
	enumName, ok := input.(string)
	if !ok {
		return pref.Value{},
			errors.Errorf("enum must be type string (%s): %s (%T)", d.FullName(), input, input)
	}
	enumNumber := d.Values().ByName((pref.Name)(enumName))
	if enumNumber == nil {
		return pref.Value{},
			errors.Errorf("enum not found (%s): %s", d.FullName(), enumName)
	}
	return pref.ValueOfEnum(enumNumber.Number()), nil
}

func convertStruct(d pref.MessageDescriptor, input interface{}) (pref.Value, error) {
	mapScan, ok := input.(map[string]interface{})
	if !ok {
		return pref.Value{},
			errors.Errorf("struct must be type map (%s): %s (%T)", d.FullName(), input, input)
	}
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(d.FullName())
	if err != nil {
		return pref.Value{}, errors.Errorf("UH OH")
	}
	message := messageType.New()
	for i := 0; i < d.Fields().Len(); i++ {
		f := d.Fields().Get(i)
		value, ok := mapScan[(string)(f.Name())]
		if !ok {
			return pref.Value{}, errors.Errorf(
				"value not found for field of %s: %s", d.FullName(), f.FullName())
		}
		parsed, err := convert(f, value)
		switch {
		case err != nil:
			return pref.Value{}, errors.Wrapf(err, "error parsing field of %s", d.FullName())
		case parsed.IsValid():
			message.Set(f, parsed)
		}
	}
	// Use a defer to copy all unmarshaled fields into the original message.
	dst :=
	defer mr.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dst.Set(fd, v)
		return true
	})
	return pref.ValueOfMessage(message), nil
}

func convertField(d pref.FieldDescriptor, input interface{}) (pref.Value, error) {
	switch parsed := input.(type) {
	case int64:
		switch d.Kind() {
		case pref.Int32Kind:
			return pref.ValueOfInt32(int32(parsed)), nil
		case pref.Int64Kind:
			return pref.ValueOfInt64(parsed), nil
		}
	case float64:
		switch d.Kind() {
		case pref.FloatKind:
			return pref.ValueOfFloat32(float32(parsed)), nil
		case pref.DoubleKind:
			return pref.ValueOfFloat64(parsed), nil
		}
	case bool:
		switch d.Kind() {
		case pref.BoolKind:
			return pref.ValueOfBool(parsed), nil
		}
	case []byte:
		return pref.Value{}, nil
	case string:
		switch d.Kind() {
		case pref.EnumKind:
			return convert(d.Enum(), input)
		case pref.StringKind:
			return pref.ValueOfString(parsed), nil
		case pref.BytesKind:
			return pref.ValueOfBytes(([]byte)(parsed)), nil
		}
	case time.Time:
		return pref.ValueOfMessage((&timestamppb.Timestamp{
			Seconds: parsed.Unix(),
			Nanos:   int32(parsed.Nanosecond()),
		}).ProtoReflect()), nil
	case nil:
		return pref.Value{}, nil
	}
	return pref.Value{}, errors.Errorf(
		"cant parse value to %s: %s (%T)", d.FullName(), input, input)
}
