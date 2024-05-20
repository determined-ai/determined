//go:build integration

package templates

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/testutils/apitest"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/templatev1"
)

func TestMain(m *testing.M) {
	pgDB, _, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestGetTemplate(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("GetTemplate that does not exist", func(t *testing.T) {
		_, err := api.GetTemplate(ctx, &apiv1.GetTemplateRequest{TemplateName: uuid.NewString()})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("GetTemplate that does exist", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		_, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)

		resp, err := api.GetTemplate(
			ctx,
			&apiv1.GetTemplateRequest{
				TemplateName: input.Name,
			},
		)
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)
	})
}

func TestGetTemplates(t *testing.T) {
	_, err := db.Bun().NewTruncateTable().Table("templates").Exec(context.Background())
	require.NoError(t, err)

	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("GetTemplates without any templates", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 0)
	})

	inputNames := []string{
		"abc",
		"bcd",
		"cde",
	}
	var inputs []*templatev1.Template
	var workspaceIDs []int32
	for _, inputName := range inputNames {
		w := &model.Workspace{
			Name:   uuid.New().String(),
			UserID: 1,
		}
		_, err := db.Bun().NewInsert().Model(w).Exec(ctx)
		require.NoError(t, err)
		workspaceIDs = append(workspaceIDs, int32(w.ID))
		input := templatev1.Template{
			Name:        inputName,
			Config:      fakeTemplate(t),
			WorkspaceId: int32(w.ID),
		}
		_, err = api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: &input})
		require.NoError(t, err)
		inputs = append(inputs, &input)
	}

	t.Run("GetTemplates with some templates", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 3)
		require.ElementsMatch(t, inputNames, templateNames(resp.Templates))
		sort.Slice(resp.Templates, func(i, j int) bool {
			return resp.Templates[i].Name < resp.Templates[j].Name
		})
		for i := 0; i < len(inputs); i++ {
			requireToJSONEq(t, inputs[i], resp.Templates[i])
		}
	})

	t.Run("GetTemplates like a name", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{Name: "b"})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 2)
		require.ElementsMatch(t, inputNames[:2], templateNames(resp.Templates))
	})

	t.Run("GetTemplates sort and order by", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{
			SortBy:  apiv1.GetTemplatesRequest_SORT_BY_NAME,
			OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 3)
		require.Equal(t, []string{"cde", "bcd", "abc"}, templateNames(resp.Templates))
	})

	t.Run("GetTemplates offset and limit", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{
			Offset: 1,
			Limit:  4,
		})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 2)
		require.Subset(t, inputNames, templateNames(resp.Templates))
	})

	t.Run("GetTemplates filter by workspace", func(t *testing.T) {
		resp, err := api.GetTemplates(ctx, &apiv1.GetTemplatesRequest{
			WorkspaceIds: []int32{workspaceIDs[0]},
		})
		require.NoError(t, err)
		require.Len(t, resp.Templates, 1)
		require.Equal(t, inputNames[0], templateNames(resp.Templates)[0])
	})
}

func TestPostTemplate(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("PostTemplate without workspace", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		}
		resp, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)
		input.WorkspaceId = model.DefaultWorkspaceID
		requireToJSONEq(t, input, resp.Template)
	})

	t.Run("PostTemplate with existing workspace", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		resp, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)
	})

	t.Run("PostTemplate with invalid workspace", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 99999,
		}
		_, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.ErrorContains(t, err, "workspace '99999' not found")
	})
}

func TestPatchTemplateConfig(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("PatchTemplateConfig that does not exist", func(t *testing.T) {
		_, err := api.PatchTemplateConfig(ctx, &apiv1.PatchTemplateConfigRequest{
			TemplateName: uuid.NewString(),
			Config:       fakeTemplate(t),
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("PatchTemplateConfig that does exist", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		}
		_, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)

		revised, err := structpb.NewStruct(map[string]any{
			"checkpoint_storage": map[string]any{
				"type":   "s3",
				"bucket": "abc",
			},
		})
		require.NoError(t, err)

		resp, err := api.PatchTemplateConfig(ctx, &apiv1.PatchTemplateConfigRequest{
			TemplateName: input.Name,
			Config:       revised,
		})
		require.NoError(t, err)
		requireToJSONEq(t, revised, resp.Template.Config)
	})
}

func TestPatchTemplateName(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("TestPatchTemplateName that doesn't exist", func(t *testing.T) {
		_, err := api.PatchTemplateName(
			ctx,
			&apiv1.PatchTemplateNameRequest{
				OldName: uuid.NewString(),
				NewName: uuid.NewString(),
			})
		require.ErrorContains(t, err, "not found")
	})
	t.Run("TestPatchTemplateName functions", func(t *testing.T) {
		// Create a template and patch name with old name.
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		resp, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)

		resp1, err := api.PatchTemplateName(ctx, &apiv1.PatchTemplateNameRequest{OldName: input.Name, NewName: input.Name})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp1.Template)

		// Create a second templates and patch name with duplicated name.
		input1 := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		_, err = api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input1})
		require.NoError(t, err)

		_, err = api.PatchTemplateName(ctx, &apiv1.PatchTemplateNameRequest{OldName: input.Name, NewName: input1.Name})
		require.ErrorContains(t, err, "templates_pkey")

		// Patch name with random name.
		randomName := uuid.NewString()
		resp1, err = api.PatchTemplateName(ctx, &apiv1.PatchTemplateNameRequest{OldName: input.Name, NewName: randomName})
		require.NoError(t, err)
		require.Equal(t, randomName, resp1.Template.Name)
	})
}

func TestPutTemplate(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("TestPutTemplate that doesn't exist", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		}
		resp, err := api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)
	})

	t.Run("TestPutTemplate that does exist", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		}
		resp, err := api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)

		revised, err := structpb.NewStruct(map[string]any{
			"checkpoint_storage": map[string]any{
				"type":   "s3",
				"bucket": "abc",
			},
		})
		require.NoError(t, err)
		input.Config = revised

		resp, err = api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input.Config, resp.Template.Config)
	})

	t.Run("TestPutTemplate with workspace change", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		resp, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)

		w := &model.Workspace{
			Name:   uuid.New().String(),
			UserID: 1,
		}
		_, err = db.Bun().NewInsert().Model(w).Exec(ctx)
		require.NoError(t, err)

		input.WorkspaceId = int32(w.ID)

		resp1, err := api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.NoError(t, err)
		require.Equal(t, resp1.Template.WorkspaceId, int32(w.ID))
	})

	t.Run("TestPutTemplate with invalid workspace change", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 1,
		}
		resp, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)
		requireToJSONEq(t, input, resp.Template)

		w := &model.Workspace{
			Name:   uuid.New().String(),
			UserID: 1,
		}
		_, err = db.Bun().NewInsert().Model(w).Exec(ctx)
		require.NoError(t, err)

		input.WorkspaceId = int32(w.ID) + 1

		_, err = api.PutTemplate(ctx, &apiv1.PutTemplateRequest{Template: input})
		require.ErrorContains(t, err, "not found")
	})
}

func TestDeleteTemplate(t *testing.T) {
	api := TemplateAPIServer{}
	ctx := apitest.WithCredentials(context.Background())

	t.Run("TestDeleteTemplate that doesn't exist", func(t *testing.T) {
		_, err := api.DeleteTemplate(ctx, &apiv1.DeleteTemplateRequest{
			TemplateName: uuid.NewString(),
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("TestDeleteTemplate that exists", func(t *testing.T) {
		input := &templatev1.Template{
			Name:        uuid.NewString(),
			Config:      fakeTemplate(t),
			WorkspaceId: 0,
		}
		_, err := api.PostTemplate(ctx, &apiv1.PostTemplateRequest{Template: input})
		require.NoError(t, err)

		_, err = api.DeleteTemplate(ctx, &apiv1.DeleteTemplateRequest{TemplateName: input.Name})
		require.NoError(t, err)

		_, err = api.GetTemplate(ctx, &apiv1.GetTemplateRequest{TemplateName: uuid.NewString()})
		require.ErrorContains(t, err, "not found")
	})
}

func fakeTemplate(t *testing.T) *structpb.Struct {
	cfg, err := structpb.NewStruct(map[string]any{
		"checkpoint_storage": map[string]any{
			"type":   "gcs",
			"bucket": uuid.NewString(),
		},
	})
	require.NoError(t, err)
	return cfg
}

func templateNames(ts []*templatev1.Template) []string {
	var names []string
	for _, t := range ts {
		names = append(names, t.Name)
	}
	return names
}

func requireToJSONEq(t *testing.T, expected, actual any, msgAndArgs ...any) {
	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err, "could not converted expected to JSON")
	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err, "could not converted actual to JSON")
	require.JSONEq(t, string(expectedJSON), string(actualJSON), msgAndArgs...)
}
