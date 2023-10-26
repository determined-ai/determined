//go:build integration
// +build integration

package internal

import (
	"sync"
	"testing"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/determined-ai/determined/proto/pkg/masterv1"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestPatchMasterConfig(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	// define a new WaitGroup that enables testing code to wait for all
	// goroutines to finish with their work
	wg := sync.WaitGroup{}

	for i := 0; i < 50; i++ {
		// increment the WaitGroup
		wg.Add(1)
		// start a new goroutine
		go func() {
			// decrement the WaitGroup
			defer wg.Done()
			_, err := api.PatchMasterConfig(ctx,
				&apiv1.PatchMasterConfigRequest{
					Config:    &masterv1.Config{Log: &masterv1.LogConfig{Level: "error", Color: true}},
					FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"log"}},
				})
			require.NoError(t, err)
		}()
	}
	// wait until all goroutines are done
	wg.Wait()
}
