package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (db *PgDB) QueryProto(queryName string, v interface{}, args ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error {
		input := make(map[string]interface{})
		if err := rows.MapScan(input); err != nil {
			return err
		}
		message, ok := val.(proto.Message)
		if !ok {
			return errors.Errorf("invalid type conversion: %T is not a Protobuf message", val)
		}
		_, err := convertStruct(message.ProtoReflect(), input)
		return err
	}
	return db.queryRows(queryName, parser, v, args...)
}

func valueOf(v proto.Message) pref.Value {
	return pref.ValueOfMessage(v.ProtoReflect())
}

func convertStruct(m pref.Message, in interface{}) (pref.Value, error) {
	md := m.Descriptor()
	switch md.FullName() {
	case "google.protobuf.Timestamp":
		switch parsed := in.(type) {
		case time.Time:
			return valueOf(&timestamppb.Timestamp{
				Seconds: parsed.Unix(),
				Nanos:   int32(parsed.Nanosecond()),
			}), nil
		}
	default:
		fds := md.Fields()
		fvs, ok := in.(map[string]interface{})
		if !ok {
			return pref.Value{}, errors.Errorf(
				"illegal conversion %T to %s: %s", in, md.FullName(), in)
		}

		for i := 0; i < fds.Len(); i++ {
			fd := fds.Get(i)
			fv, ok := fvs[(string)(fd.Name())]
			if !ok || (fd.IsWeak() && fd.Message().IsPlaceholder()) {
				continue
			}

			switch fd.Cardinality() {
			case pref.Repeated:
			default:
				v, err := convertField(m.NewField(fd), fd, fv)
				if err != nil {
					return pref.Value{}, errors.Wrapf(err, "error parsing %s", fd.Name())
				}
				m.Set(fd, v)
			}

		}
		return pref.ValueOf(m), nil
	}
	return pref.Value{}, errors.Errorf(
		"illegal conversion %T to %s: %s", in, md.FullName(), in)
}

func convertField(v pref.Value, d pref.FieldDescriptor, in interface{}) (pref.Value, error) {
	switch d.Kind() {
	case pref.BoolKind:
		switch parsed := in.(type) {
		case bool:
			return pref.ValueOfBool(parsed), nil
		}
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		switch parsed := in.(type) {
		case int64:
			return pref.ValueOfInt32(int32(parsed)), nil
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		switch parsed := in.(type) {
		case int64:
			return pref.ValueOfInt64(parsed), nil
		}
	case pref.FloatKind:
		switch parsed := in.(type) {
		case float64:
			return pref.ValueOfFloat32(float32(parsed)), nil
		}
	case pref.DoubleKind:
		switch parsed := in.(type) {
		case float64:
			return pref.ValueOfFloat64(parsed), nil
		}
	case pref.StringKind:
		switch parsed := in.(type) {
		case string:
			return pref.ValueOfString(parsed), nil
		}
	case pref.BytesKind:
		switch parsed := in.(type) {
		case []byte:
			return pref.ValueOfBytes(parsed), nil
		}
	case pref.EnumKind:
		switch parsed := in.(type) {
		case string:
			enumNumber := d.Enum().Values().ByName((pref.Name)(parsed))
			if enumNumber == nil {
				return pref.Value{},
					errors.Errorf("enum not found (%s): %s", d.FullName(), parsed)
			}
			return pref.ValueOfEnum(enumNumber.Number()), nil
		}
	case pref.MessageKind, pref.GroupKind:
		return convertStruct(v.Message(), in)
	}
	return pref.Value{}, errors.Errorf(
		"illegal conversion %T to %s: %s", in, d.FullName(), in)
}
