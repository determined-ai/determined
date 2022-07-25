package actor

import (
	"reflect"
	"testing"
)

func TestAddr(t *testing.T) {
	type args struct {
		path []string
	}
	type testCase struct {
		name   string
		args   args
		want   Address
		panics bool
	}
	tests := []testCase{
		{"one path", args{path: []string{"test"}}, Address{"/test"}, false},
		{"multiple path", args{path: []string{"test", "two"}}, Address{"/test/two"}, false},
		{"illegal character", args{path: []string{"test", "%"}}, Address{"/test/%"}, true},
		{"no address", args{path: []string{}}, Address{}, true},
		{"with slash", args{path: []string{"test/"}}, Address{}, true},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); tc.panics && r == nil {
					t.Errorf("The code did not panic")
				} else if !tc.panics && r != nil {
					panic(r)
				}
			}()
			path := make([]interface{}, 0, len(tc.args.path))
			for _, v := range tc.args.path {
				path = append(path, v)
			}
			if got := Addr(path...); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Addr() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_String(t *testing.T) {
	type fields struct {
		path string
	}
	type testCase struct {
		name   string
		fields fields
		want   string
	}
	tests := []testCase{
		{"one path", fields{path: "/test"}, "/test"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			a := Address{
				path: tc.fields.path,
			}
			if got := a.String(); got != tc.want {
				t.Errorf("Address.String() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_Parent(t *testing.T) {
	type fields struct {
		path string
	}
	type testCase struct {
		name   string
		fields fields
		want   Address
	}
	tests := []testCase{
		{"parent", fields{path: "/test/child"}, Addr("test")},
		{"to root", fields{path: "/test"}, Address{path: "/"}},
		{"root", fields{path: "/"}, Address{path: "/"}},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			a := Address{
				path: tc.fields.path,
			}
			if got := a.Parent(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Address.Parent() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_Child(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		child string
	}
	type testCase struct {
		name   string
		fields fields
		args   args
		want   Address
		panics bool
	}
	tests := []testCase{
		{"child", fields{path: "/test"}, args{child: "child"}, Address{path: "/test/child"}, false},
		{"child", fields{path: "/test"}, args{child: "child/"}, Address{path: "/test/child"}, true},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); tc.panics && r == nil {
					t.Errorf("The code did not panic")
				} else if !tc.panics && r != nil {
					panic(r)
				}
			}()
			a := Address{
				path: tc.fields.path,
			}
			if got := a.Child(tc.args.child); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Address.Child() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_Local(t *testing.T) {
	type fields struct {
		path string
	}
	type testCase struct {
		name   string
		fields fields
		want   string
	}
	tests := []testCase{
		{"root", fields{path: "/test"}, "test"},
		{"with child", fields{path: "/test/child"}, "child"},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			a := Address{
				path: tc.fields.path,
			}
			if got := a.Local(); got != tc.want {
				t.Errorf("Address.Local() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_IsAncestorOf(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		address Address
	}
	type testCase struct {
		name   string
		fields fields
		args   args
		want   bool
	}
	tests := []testCase{
		{"is ancestor", fields{path: "/test"}, args{Addr("test", "child")}, true},
		{"opposite ancestor", fields{path: "/test/child"}, args{Addr("test")}, false},
		{"equal", fields{path: "/test/child"}, args{Addr("test", "child")}, false},
		{"is not ancestor", fields{path: "/test/child1"}, args{Addr("test", "child2")}, false},
		{"common parent", fields{path: "/test/nest1"}, args{Addr("test", "nest2", "child2")}, false},
		{"close prefix", fields{path: "/a"}, args{Addr("ab")}, false},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			a := Address{
				path: tc.fields.path,
			}
			if got := a.IsAncestorOf(tc.args.address); got != tc.want {
				t.Errorf("Address.IsAncestorOf() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestAddress_nextParent(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		address Address
	}
	type testCase struct {
		name   string
		fields fields
		args   args
		want   Address
		panics bool
	}
	tests := []testCase{
		{
			"next parent",
			fields{path: "/grand"},
			args{Addr("grand", "parent", "child")},
			Addr("grand", "parent"), false,
		},
		{
			"last parent",
			fields{path: "/test"},
			args{Addr("test", "child")},
			Addr("test", "child"), false,
		},
		{
			"is not ancestor",
			fields{path: "/test/child2"},
			args{Addr("test", "child")},
			Addr("test", "child"), true,
		},
		{
			"from root",
			fields{path: "/"},
			args{Addr("test")},
			Addr("test"), false,
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); tc.panics && r == nil {
					t.Errorf("The code did not panic")
				} else if !tc.panics && r != nil {
					panic(r)
				}
			}()
			a := Address{
				path: tc.fields.path,
			}
			if got := a.nextParent(tc.args.address); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Address.nextParent() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
