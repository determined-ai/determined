package api

import (
	"reflect"
	"testing"
)

type parsed struct {
	Value string `json:"value"`
}

type args struct {
	raw     []byte
	msgType reflect.Type
}

type testCase struct {
	name    string
	args    args
	want    interface{}
	wantErr bool
}

var blob = []byte(`{"value": "test"}`)

func Test_parseMsg(t *testing.T) {
	tests := []testCase{
		{"pointer", args{raw: blob, msgType: reflect.TypeOf(&parsed{})}, &parsed{Value: "test"}, false},
		{"value", args{raw: blob, msgType: reflect.TypeOf(parsed{})}, parsed{Value: "test"}, false},
		{"err", args{raw: []byte("[]"), msgType: reflect.TypeOf(parsed{})}, nil, true},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMsg(tc.args.raw, tc.args.msgType)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseMsg() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseMsg() = %v, want %v", got, tc.want)
			}
		})
	}
	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
