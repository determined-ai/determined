package internal

import (
	"fmt"
	"reflect"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type apiServer struct {
	m *Master
}

// paginate returns a paginated subset of the values and sets the pagination response.
func (a *apiServer) paginate(p **apiv1.Pagination, values interface{}, offset, limit int32) error {
	rv := reflect.ValueOf(values)
	if rv.Elem().Kind() != reflect.Slice {
		return errors.Errorf("error paginating non-slice type: %T", rv.Kind())
	}
	total := int32(rv.Elem().Len())
	pagination, err := api.Paginate(int(total), int(offset), int(limit))
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	*p = &apiv1.Pagination{
		Offset:     offset,
		Limit:      limit,
		StartIndex: int32(pagination.StartIndex),
		EndIndex:   int32(pagination.EndIndex),
		Total:      total,
	}
	rv.Elem().Set(rv.Elem().Slice(pagination.StartIndex, pagination.EndIndex))
	return nil
}

// sort sorts the provided slice in place. The second parameter denotes whether sorting should be
// in ascending or descending order. All following parameters are the sort keys. Sort keys must be
// the same value as the field number that must be sorted.
func (a *apiServer) sort(
	slice interface{}, order apiv1.OrderBy, keys ...interface{}) {
	rv := reflect.ValueOf(slice)
	sort.Slice(slice, func(i, j int) bool {
		a1 := rv.Index(i).Interface().(proto.Message).ProtoReflect()
		a2 := rv.Index(j).Interface().(proto.Message).ProtoReflect()
		if order == apiv1.OrderBy_ORDER_BY_DESC {
			a2, a1 = a1, a2
		}
		d1, d2 := a1.Descriptor(), a2.Descriptor()
		for _, key := range keys {
			key := reflect.ValueOf(key).Int()
			if key == 0 {
				continue
			}
			fn := pref.FieldNumber(key)
			fd1, fd2 := d1.Fields().ByNumber(fn), d2.Fields().ByNumber(fn)
			f1, f2 := a1.Get(fd1), a2.Get(fd2)

			if fd1.Cardinality() == pref.Repeated {
				panic(fmt.Sprintf("incomparable cardinality for field: %s", fd1.FullName()))
			}
			switch fd1.Kind() {
			case pref.BoolKind:
				v1, v2 := f1.Bool(), f2.Bool()
				if v1 == v2 {
					continue
				}
				return v1
			case pref.EnumKind:
				v1, v2 := f1.Enum(), f2.Enum()
				if v1 == v2 {
					continue
				}
				return v1 < v2
			case pref.Int32Kind, pref.Sint32Kind, pref.Int64Kind,
				pref.Sint64Kind, pref.Sfixed32Kind, pref.Sfixed64Kind:
				v1, v2 := f1.Int(), f2.Int()
				if v1 == v2 {
					continue
				}
				return v1 < v2
			case pref.Uint32Kind, pref.Uint64Kind, pref.Fixed32Kind, pref.Fixed64Kind:
				v1, v2 := f1.Uint(), f2.Uint()
				if v1 == v2 {
					continue
				}
				return v1 < v2
			case pref.FloatKind, pref.DoubleKind:
				v1, v2 := f1.Float(), f2.Float()
				if v1 == v2 {
					continue
				}
				return v1 < v2
			case pref.StringKind:
				v1, v2 := f1.String(), f2.String()
				if v1 == v2 {
					continue
				}
				return v1 < v2
			case pref.MessageKind:
				v1, v2 := f1.Message().Interface(), f2.Message().Interface()
				switch {
				case v1 == nil && v2 == nil:
					continue
				case v1 == nil:
					return true
				case v2 == nil:
					return false
				}
				switch t1 := v1.(type) {
				case *timestamppb.Timestamp:
					t2 := v2.(*timestamppb.Timestamp)
					if t1.Seconds == t2.Seconds {
						if t1.Nanos == t2.Nanos {
							continue
						}
						return t1.Nanos < t2.Nanos
					}
					return t1.Seconds < t2.Seconds
				default:
					panic(fmt.Sprintf("incomparable message type: %T", t1))
				}
			default:
				panic(fmt.Sprintf("incomparable field type for %s: %s", fd1.FullName(), fd1.Kind()))
			}
		}
		return false
	})
}

// filter filters in place the provide reference to the slice. The check function is given an
// index of the current element it will check to filter. Returning false will remove the
// element from the slice.
func (a *apiServer) filter(values interface{}, check func(int) bool) {
	rv := reflect.ValueOf(values)
	results := reflect.MakeSlice(rv.Type().Elem(), 0, 0)
	for i := 0; i < rv.Elem().Len(); i++ {
		if check(i) {
			results = reflect.Append(results, rv.Elem().Index(i))
		}
	}
	rv.Elem().Set(results)
}

func (a *apiServer) actorRequest(addr string, req actor.Message, v interface{}) error {
	actorAddr := actor.Address{}
	if err := actorAddr.UnmarshalText([]byte(addr)); err != nil {
		return status.Errorf(codes.InvalidArgument, "/api/v1%s is not a valid path", addr)
	}
	resp := a.m.system.AskAt(actorAddr, req)
	if err := api.ProcessActorResponseError(&resp); err != nil {
		return err
	}
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
	return nil
}

// askAtDefaultSystem asks addr the req and puts the response into what v points at.
func (a *apiServer) askAtDefaultSystem(
	addr actor.Address, req interface{}, v interface{},
) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return status.Errorf(
			codes.Internal,
			`ask to actor %s contains valid but unsettable response holder %T`, addr, v,
		)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := a.m.system.AskAt(addr, req); {
	case resp.Source() == nil:
		return status.Errorf(
			codes.NotFound,
			"actor %s could not be found", addr,
		)
	case expectingResponse && resp.Empty(), expectingResponse && resp.Get() == nil:
		return status.Errorf(
			codes.NotFound,
			"actor %s did not respond", addr,
		)
	case resp.Error() != nil:
		switch {
		case errors.Is(resp.Error(), api.ErrBadRequest):
			return status.Errorf(
				codes.InvalidArgument,
				resp.Error().Error(),
			)
		case errors.Is(resp.Error(), api.ErrNotFound):
			return status.Errorf(
				codes.NotFound,
				resp.Error().Error(),
			)
		default:
			return status.Errorf(
				codes.Internal,
				"actor %s returned error: %s", addr, resp.Error(),
			)
		}
	default:
		if expectingResponse {
			if reflect.ValueOf(v).Elem().Type() != reflect.ValueOf(resp.Get()).Type() {
				return unexpectedMessageError(addr, resp)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}
