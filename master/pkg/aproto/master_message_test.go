package aproto

import (
	"cmp"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/stretchr/testify/require"
)

func TestContainerStarted_Addresses(t *testing.T) {
	tests := []struct {
		name  string
		input ContainerStarted
		want  []cproto.Address
	}{
		{
			name:  "ipv4 docker and agent addrs",
			input: mustUnmarshal[ContainerStarted](t, mustReadFile(t, "testdata/ipv4_only.input.json")),
			want:  mustUnmarshal[[]cproto.Address](t, mustReadFile(t, "testdata/ipv4_only.output.json")),
		},
		{
			name:  "ipv4 and ipv6",
			input: mustUnmarshal[ContainerStarted](t, mustReadFile(t, "testdata/ipv6_ipv4_mix.input.json")),
			want:  mustUnmarshal[[]cproto.Address](t, mustReadFile(t, "testdata/ipv6_ipv4_mix.output.json")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Addresses()
			slices.SortFunc(got, cmpAddress)
			slices.SortFunc(tt.want, cmpAddress)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ContainerStarted.Addresses() = %s, want %v", got, tt.want)
			}
		})
	}
}

func cmpAddress(x, y cproto.Address) int {
	return cmp.Compare(x.String(), y.String())
}

func mustReadFile(t *testing.T, name string) []byte {
	bs, err := os.ReadFile(name)
	require.NoError(t, err, "failed to read file %s", name)
	return bs
}

func mustUnmarshal[T any](t *testing.T, input []byte) T {
	var out T
	err := json.Unmarshal(input, &out)
	require.NoError(t, err, "failed to marshal into %T", out)
	return out
}
