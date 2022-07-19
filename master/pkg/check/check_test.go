package check

import (
	"testing"

	"gotest.tools/assert"
)

func TestTrue(t *testing.T) {
	type args struct {
		condition  bool
		msgAndArgs []interface{}
	}
	type testCase struct {
		name string
		args args
		msg  string
	}
	tests := []testCase{
		{"true", args{condition: true}, ""},
		{"false", args{condition: false}, "expected true, got false"},
		{
			"customMsg",
			args{condition: false, msgAndArgs: []interface{}{"failure"}},
			"failure: expected true, got false",
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			err := True(tc.args.condition, tc.args.msgAndArgs...)
			if tc.msg == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, tc.msg)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestFalse(t *testing.T) {
	type args struct {
		condition  bool
		msgAndArgs []interface{}
	}
	type testCase struct {
		name string
		args args
		msg  string
	}
	tests := []testCase{
		{"false", args{condition: false}, ""},
		{"true", args{condition: true}, "expected false, got true"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			err := False(tc.args.condition, tc.args.msgAndArgs...)
			if tc.msg == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, tc.msg)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestEqual(t *testing.T) {
	type args struct {
		actual     interface{}
		expected   interface{}
		msgAndArgs []interface{}
	}
	type testCase struct {
		name string
		args args
		msg  string
	}
	tests := []testCase{
		{"equal", args{actual: 3, expected: 3}, ""},
		{
			"notEqual",
			args{actual: "a", expected: "b"},
			"a does not equal b",
		},
		{
			"notEqualTypes",
			args{actual: 3, expected: 3.0},
			"3 does not equal 3",
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			err := Equal(tc.args.actual, tc.args.expected, tc.args.msgAndArgs...)
			if tc.msg == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, tc.msg)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestGreaterThanOrEqualTo(t *testing.T) {
	type args struct {
		actual     int
		expected   int
		msgAndArgs []interface{}
	}
	type testCase struct {
		name string
		args args
		msg  string
	}
	tests := []testCase{
		{"greater", args{actual: 3, expected: 2}, ""},
		{"equal", args{actual: 2, expected: 2}, ""},
		{"less", args{actual: 1, expected: 2}, "1 is not greater than or equal to 2"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			err := GreaterThanOrEqualTo(tc.args.actual, tc.args.expected, tc.args.msgAndArgs...)
			if tc.msg == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, tc.msg)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
