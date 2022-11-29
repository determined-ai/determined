package tasklist

import (
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/determined-ai/determined/master/pkg/actor"
)

var nilActor = actor.ActorFunc(func(context *actor.Context) error {
	return nil
})

func TestAllocationRequestComparator(t *testing.T) {
	newTime := time.Now()
	oldTime := newTime.Add(-time.Minute * 15)

	system := actor.NewSystem("test")
	r1 := system.MustActorOf(actor.Addr("r1"), nilActor)
	r2 := system.MustActorOf(actor.Addr("r2"), nilActor)

	type args struct {
		a *sproto.AllocateRequest
		b *sproto.AllocateRequest
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "old tasks first",
			args: args{
				a: &sproto.AllocateRequest{
					TaskID:            "task1",
					JobID:             "job1",
					JobSubmissionTime: oldTime,
				},
				b: &sproto.AllocateRequest{
					TaskID:            "task2",
					JobID:             "job2",
					JobSubmissionTime: newTime,
				},
			},
			want: -1,
		},
		{
			name: "new tasks last",
			args: args{
				a: &sproto.AllocateRequest{
					TaskID:            "task1",
					JobID:             "job1",
					JobSubmissionTime: newTime,
				},
				b: &sproto.AllocateRequest{
					TaskID:            "task2",
					JobID:             "job2",
					JobSubmissionTime: oldTime,
				},
			},
			want: 1,
		},
		{
			name: "actor registration breaks tie",
			args: args{
				a: &sproto.AllocateRequest{
					TaskID:            "task1",
					JobID:             "job1",
					JobSubmissionTime: newTime,
					AllocationRef:     r1,
				},
				b: &sproto.AllocateRequest{
					TaskID:            "task2",
					JobID:             "job2",
					JobSubmissionTime: newTime,
					AllocationRef:     r2,
				},
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allocationRequestComparator(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("allocationRequestComparator() = %v, want %v", got, tt.want)
			}
		})
	}
}
