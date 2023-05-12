package dispatcherrm

import (
	"testing"

	"github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
)

func Test_isForeignJob(t *testing.T) {
	type args struct {
		v launcher.DispatchInfo
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Not a dispatch RM job",
			args: args{
				v: launcher.DispatchInfo{
					LaunchedCapsuleReference: &launcher.OwnedResourceReference{
						Name: launcher.PtrString("something"),
					},
				},
			},
			want: true,
		},
		{
			name: "Is a dispatch RM job",
			args: args{
				v: launcher.DispatchInfo{
					LaunchedCapsuleReference: &launcher.OwnedResourceReference{
						Name: launcher.PtrString("det"),
					},
				},
			},
			want: false,
		},
		{
			name: "Is another RM job",
			args: args{
				v: launcher.DispatchInfo{
					LaunchedCapsuleReference: &launcher.OwnedResourceReference{
						Name: launcher.PtrString("DAI-HPC-Resources"),
					},
				},
			},
			want: false,
		},
		{
			name: "Is yet another dispatch RM job",
			args: args{
				v: launcher.DispatchInfo{
					LaunchedCapsuleReference: &launcher.OwnedResourceReference{
						Name: launcher.PtrString("DAI-HPC-Queues"),
					},
				},
			},
			want: false,
		},
		{
			name: "nil pointer test",
			args: args{
				v: launcher.DispatchInfo{
					LaunchedCapsuleReference: &launcher.OwnedResourceReference{
						Name: nil,
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isForeignJob(tt.args.v); got != tt.want {
				t.Errorf("isForeignJob() = %v, want %v", got, tt.want)
			}
		})
	}
}
