//go:build integration

package templates

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/test/testutils/apitest"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

func TestUnmarshalTemplateConfig(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	u, err := user.ByUsername(ctx, "determined")
	require.NoError(t, err)

	t.Run("UnmarshalTemplateConfig that does not exist", func(t *testing.T) {
		var m map[string]any
		err = UnmarshalTemplateConfig(ctx, uuid.NewString(), u, &m, false)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("UnmarshalTemplateConfig that does exist", func(t *testing.T) {
		cfgBucket := uuid.NewString()
		cfg, err := structpb.NewStruct(map[string]any{
			"checkpoint_storage": map[string]any{
				"type":   "gcs",
				"bucket": cfgBucket,
			},
		})
		require.NoError(t, err)

		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      cfg,
			WorkspaceId: 0,
		}
		resp, err := api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)

		fakeConfig := schemas.WithDefaults(expconf.ExperimentConfigV0{
			RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
				RawSharedFSConfig: &expconf.SharedFSConfigV0{
					RawHostPath: ptrs.Ptr("/home/ckpts"),
				},
			},
			RawEntrypoint: &expconf.EntrypointV0{
				RawEntrypoint: ptrs.Ptr("model.Classifier"),
			},
			RawHyperparameters: map[string]expconf.HyperparameterV0{
				"global_batch_size": {
					RawConstHyperparameter: &expconf.ConstHyperparameterV0{
						RawVal: ptrs.Ptr(1),
					},
				},
			},
			RawSearcher: &expconf.SearcherConfigV0{
				RawSingleConfig: &expconf.SingleConfigV0{
					RawMaxLength: &expconf.LengthV0{
						Unit:  expconf.Batches,
						Units: 1,
					},
				},
				RawMetric: ptrs.Ptr("loss_of_something"),
			},
		})
		err = UnmarshalTemplateConfig(ctx, input.Name, u, &fakeConfig, false)
		require.NoError(t, err)
		require.NotNil(t, fakeConfig.CheckpointStorage().RawGCSConfig)
		require.Equal(t, cfgBucket, fakeConfig.CheckpointStorage().RawGCSConfig.Bucket())
	})
}

func TestDeleteWorkspaceTemplates(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	_, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{
		Template: &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		},
	})
	require.NoError(t, err)

	n, err := workspaceTemplatesCount(ctx, 1)
	require.NoError(t, err)
	require.Greater(t, n, 0)

	err = DeleteWorkspaceTemplates(ctx, 1)
	require.NoError(t, err)

	n, err = workspaceTemplatesCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func workspaceTemplatesCount(ctx context.Context, workspaceID int) (int, error) {
	return db.Bun().NewSelect().Table("templates").Where("workspace_id = ?", workspaceID).Count(ctx)
}
