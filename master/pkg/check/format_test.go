package check

import (
	"testing"
)

func Test_messageFromMsgAndArgs(t *testing.T) {
	type args struct {
		msgAndArgs []interface{}
	}
	type testCase struct {
		name string
		args args
		want string
	}
	tests := []testCase{
		{"Empty", args{[]interface{}{}}, ""},
		{"Single", args{[]interface{}{"string"}}, "string"},
		{"Int", args{[]interface{}{5}}, "5"},
		{"Struct", args{[]interface{}{struct {
			Value string
		}{"test"}}}, "{Value:test}"},
		{"Format", args{[]interface{}{"value %d", 5}}, "value 5"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			if got := messageFromMsgAndArgs(false, tc.args.msgAndArgs...); got != tc.want {
				t.Errorf("messageFromMsgAndArgs() = %v, want %v", got, tc.want)
			}
		})
	}
	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func Test_format(t *testing.T) {
	value := 4
	ptr := &value
	ptrptr := &ptr

	type testCase struct {
		name string
		arg  interface{}
		want string
	}
	tests := []testCase{
		{"value", value, "4"},
		{"ptr", ptr, "*int(4)"},
		{"ptrptr", ptrptr, "**int(4)"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			if got := format(tc.arg); got != tc.want {
				t.Errorf("format() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
