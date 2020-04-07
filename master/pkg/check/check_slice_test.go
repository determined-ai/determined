package check

import (
	"testing"
)

func TestContains(t *testing.T) {
	type args struct {
		actual     interface{}
		expected   []interface{}
		msgAndArgs []interface{}
	}
	type testCase struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []testCase{
		{"nil value", args{expected: []interface{}{nil}}, false},
		{"nil list", args{}, true},
		{"contains", args{actual: 1, expected: []interface{}{0, 1, 2, 3}}, false},
		{"not contains", args{actual: 1, expected: []interface{}{0, 2, 3}}, true},
	}

	runTestCase := func(t *testing.T, tt testCase) {
		t.Run(tt.name, func(t *testing.T) {
			err := Contains(tt.args.actual, tt.args.expected, tt.args.msgAndArgs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Contains() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	for _, tt := range tests {
		runTestCase(t, tt)
	}
}
