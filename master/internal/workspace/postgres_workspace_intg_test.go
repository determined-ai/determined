//go:build integration
// +build integration

package workspace

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

const (
	cluster1 = "C1"
	cluster2 = "C2"
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

type Bindings struct {
	bun.BaseModel `bun:"table:workspace_namespace_bindings"`
	WorkspaceID   int    `bun:"workspace_id"`
	ClusterName   string `bun:"cluster_name"`
	NamespaceName string `bun:"namespace"`
}

func TestGetNamespaceFromWorkspace(t *testing.T) {
	ctx := context.Background()
	wkspID1, wksp1 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID2, wksp2 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	_, wksp3 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	b1 := Bindings{WorkspaceID: wkspID1, ClusterName: "C1", NamespaceName: "n1"}
	b2 := Bindings{WorkspaceID: wkspID1, ClusterName: "C2", NamespaceName: "n2"}
	b3 := Bindings{WorkspaceID: wkspID2, ClusterName: "C1", NamespaceName: "n3"}

	bindings := []Bindings{b1, b2, b3}

	_, err := db.Bun().NewInsert().Model(&bindings).Exec(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		workspaceName string
		clusterName   string
		namespaceName string
		wantErr       bool
	}{
		{"insert-and-check-1", wksp1, "C1", "n1", true},
		{"insert-and-check-2", wksp1, "C2", "n2", true},
		{"insert-and-check-3", wksp2, "C1", "n3", true},
		{"unknown-binding-1", wksp2, "C2", "", false},
		{"unknown-binding-2", wksp3, "C1", "", false},
		{"unknown-wksp", "test-unknown-wksp", "C1", "", false},
		{"unknown-cluster", wksp2, "test-unknown-cluster", "", false},
		{"unknown-wksp-cluster", "test-unknown-wksp", "test-unknown-cluster", "", false},
	}

	for _, tt := range tests {
		ns, err := GetNamespaceFromWorkspace(ctx, tt.workspaceName, tt.clusterName)
		if tt.wantErr {
			require.NoError(t, err)
			require.Equal(t, ns, tt.namespaceName)
		} else {
			require.ErrorContains(t, err, "no rows")
		}
	}
}

func TestGetAllNamespacesForRM(t *testing.T) {
	ctx := context.Background()
	wkspID1, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID2, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	b1 := Bindings{WorkspaceID: wkspID1, ClusterName: cluster1, NamespaceName: "n1"}
	b2 := Bindings{WorkspaceID: wkspID1, ClusterName: cluster2, NamespaceName: "n2"}
	b3 := Bindings{WorkspaceID: wkspID2, ClusterName: cluster1, NamespaceName: "n3"}

	bindings := []Bindings{b1, b2, b3}

	_, err := db.Bun().NewInsert().Model(&bindings).Exec(ctx)
	require.NoError(t, err)

	ns, err := GetAllNamespacesForRM(ctx, cluster1)
	require.NoError(t, err)
	require.Equal(t, ns, []string{"n1", "n3"})

	ns, err = GetAllNamespacesForRM(ctx, cluster2)
	require.NoError(t, err)
	require.Equal(t, ns, []string{"n2"})

	ns, err = GetAllNamespacesForRM(ctx, "cluster3")
	require.NoError(t, err)
	require.Equal(t, ns, []string(nil))
}
