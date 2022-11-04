package containers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/options"
)

func TestAddProxyInfo(t *testing.T) {
	type args struct {
		env  []string
		opts options.AgentOptions
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "add proxy",
			args: args{
				env: []string{
					"FIRST_VAR=1",
				},
				opts: options.AgentOptions{
					HTTPProxy:  "192.168.1.1",
					HTTPSProxy: "192.168.1.2",
					FTPProxy:   "192.168.1.3",
					NoProxy:    "*.test.com",
				},
			},
			want: []string{
				"FIRST_VAR=1",
				"HTTP_PROXY=192.168.1.1",
				"HTTPS_PROXY=192.168.1.2",
				"FTP_PROXY=192.168.1.3",
				"NO_PROXY=*.test.com",
			},
		},
		{
			name: "no add proxy",
			args: args{
				env: []string{
					"FIRST_VAR=1",
				},
				opts: options.AgentOptions{},
			},
			want: []string{
				"FIRST_VAR=1",
			},
		},
		{
			name: "already added proxy",
			args: args{
				env: []string{
					"FIRST_VAR=1",
					"HTTP_PROXY=10.0.0.1",
				},
				opts: options.AgentOptions{
					HTTPProxy: "10.0.0.2",
				},
			},
			want: []string{
				"FIRST_VAR=1",
				"HTTP_PROXY=10.0.0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ElementsMatch(
				t,
				tt.want,
				addProxyInfo(tt.args.env, tt.args.opts),
			)
		})
	}
}
