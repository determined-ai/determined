package check

import (
	"fmt"
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

func TestIsValidK8sLabel(t *testing.T) {
	tests := []struct {
		label   string
		wantErr bool
	}{
		{"valid-label", false},
		{"Valid-Label_123", false},
		{"a", false},
		{"a1", false},
		{"1a", false},
		{"a_b.c", false},
		{"-invalid", true},
		{"invalid-", true},
		{"_invalid", true},
		{"invalid_", true},
		{".invalid", true},
		{"invalid.", true},
		{"", true},
		{"this-label-is-way-too-long-and-should-definitely-fail-because-it-is-over-sixty-three-characters", true},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			err := IsValidK8sLabel(tt.label)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidK8sLabel(%s) error = %v, wantErr %v", tt.label, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidIPV4(t *testing.T) {
	tests := []struct {
		ip      string
		wantErr bool
	}{
		{"192.168.1.1", false},
		{"0.0.0.0", false},
		{"255.255.255.255", false},
		{"1.1.1.1", false},
		{"192.168.1", true},
		{"192.168.1.256", true},
		{"192.168.1.-1", true},
		{"192.168.1.1.1", true},
		{"192.168.1.01", false},
		{"invalid_ip", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			err := IsValidIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidIPV4(%s) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestBetweenInclusive(t *testing.T) {
	tests := []struct {
		actual  interface{}
		lower   interface{}
		upper   interface{}
		wantErr bool
	}{
		{10, 5, 15, false},
		{10, 10, 15, false},
		{10, 5, 10, false},
		{10, 10, 10, false},
		{10, 11, 15, true},
		{10, 5, 9, true},
		{10, "5", 15, true},
		{int32(10), int32(5), int32(15), false},
		{int32(10), int32(10), int32(15), false},
		{int32(10), int32(5), int32(10), false},
		{int32(10), int32(10), int32(10), false},
		{int32(10), int32(11), int32(15), true},
		{int32(10), int32(5), int32(9), true},
		{int32(10), 5, int32(15), true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v between %v and %v", tt.actual, tt.lower, tt.upper), func(t *testing.T) {
			err := BetweenInclusive(tt.actual, tt.lower, tt.upper)
			if (err != nil) != tt.wantErr {
				t.Errorf("BetweenInclusive(%v, %v, %v) error = %v, wantErr %v", tt.actual, tt.lower, tt.upper, err, tt.wantErr)
			}
		})
	}
}
