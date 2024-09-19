//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
)

func TestGetSearchConfig(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	exp := createTestExp(t, api, curUser)
	expectedBytes, err := db.SingleDB().ExperimentConfigRaw(exp.ID)
	require.NoError(t, err)
	expected := make(map[string]any)
	require.NoError(t, json.Unmarshal(expectedBytes, &expected))

	resp, err := api.GetSearch(ctx, &apiv2.GetSearchRequest{
		SearchId: int32(exp.ID),
	})
	require.NoError(t, err)

	cases := []struct {
		name   string
		config *structpb.Struct
	}{
		{"GetSearchResponse.Config", resp.Config},
		{"GetSearchResponse.Search.Config", resp.Search.Config}, //nolint:staticcheck
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, expected, c.config.AsMap())
		})
	}
}
