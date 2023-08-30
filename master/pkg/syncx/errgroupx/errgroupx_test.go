package errgroupx

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrgroupxCancelByReturnError(t *testing.T) {
	ctx := context.Background()
	g := WithContext(ctx)

	finished := make(chan bool, 1)
	g.Go(func(ctx context.Context) error {
		return fmt.Errorf("non nil error")
	})
	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		finished <- true
		return nil
	})

	require.Error(t, g.Wait())
	require.True(t, <-finished)
	require.NoError(t, ctx.Err()) // Original context not canceled.
}

func TestErrgroupxCancelingParentCancels(t *testing.T) {
	for _, cancelGroup := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		g := WithContext(ctx)

		finished := make(chan bool, 1)
		g.Go(func(ctx context.Context) error {
			<-ctx.Done()
			finished <- true
			return nil
		})

		if cancelGroup {
			g.cancel()
			require.NoError(t, ctx.Err())
		} else {
			cancel()
			require.Error(t, ctx.Err())
		}
		require.True(t, <-finished)
		require.NoError(t, g.Wait())
		cancel()
	}
}

func TestErrgroupxRecover(t *testing.T) {
	eg := WithContext(context.Background()).WithRecover()
	eg.Go(func(ctx context.Context) error {
		panic("oh no")
	})
	err := eg.Wait()
	t.Log(err)
	require.ErrorContains(t, err, "oh no")
}
