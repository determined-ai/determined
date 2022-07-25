package actor

import "testing"

func Test_errUnexpectedMessage_Error(t *testing.T) {
	type fields struct {
		ctx *Context
	}
	type testCase struct {
		name   string
		fields fields
		want   string
	}
	tests := []testCase{
		{
			"blank message",
			fields{ctx: &Context{message: ""}},
			"unexpected message from <external> to <unknown> (string):  (no response expected)",
		},
		{
			"sender",
			fields{ctx: &Context{message: "msg", sender: &Ref{address: Address{"/test"}}}},
			"unexpected message from /test to <unknown> (string): msg (no response expected)",
		},
		{
			"recipient",
			fields{ctx: &Context{message: "msg", recipient: &Ref{address: Address{"/test"}}}},
			"unexpected message from <external> to /test (string): msg (no response expected)",
		},
		{
			"response",
			fields{ctx: &Context{message: "msg", result: make(chan Message)}},
			"unexpected message from <external> to <unknown> (string): msg (response expected)",
		},
		{
			"array",
			fields{ctx: &Context{message: []string{"array"}}},
			"unexpected message from <external> to <unknown> ([]string): [array] (no response expected)",
		},
		{
			"struct",
			fields{ctx: &Context{message: struct{ key string }{key: "value"}}},
			"unexpected message from <external> to <unknown> " +
				"(struct { key string }): {key:value} (no response expected)",
		},
		{
			"pointer",
			fields{ctx: &Context{message: &struct{ key string }{key: "value"}}},
			"unexpected message from <external> to <unknown> " +
				"(*struct { key string }): {key:value} (no response expected)",
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			e := errUnexpectedMessage{
				ctx: tc.fields.ctx,
			}
			if got := e.Error(); got != tc.want {
				t.Errorf("errUnexpectedMessage.Error() = %v, want %v", got, tc.want)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestErrUnexpectedMessage(t *testing.T) {
	type args struct {
		ctx *Context
	}
	type testCase struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []testCase{
		{"error", args{ctx: &Context{}}, true},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			if err := ErrUnexpectedMessage(tc.args.ctx); (err != nil) != tc.wantErr {
				t.Errorf("ErrUnexpectedMessage() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
