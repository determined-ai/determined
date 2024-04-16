//go:build integration
// +build integration

package saml

import (
	"context"
	"encoding/xml"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/crewjam/saml"
)

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../../static/migrations")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}
	os.Exit(m.Run())
}

func TestSAMLWorkflowAutoProvision(t *testing.T) {
	// First, make sure the mock SAML service is created.
	s := mockService(true)
	require.NotNil(t, s)

	ctx := context.Background()

	username := uuid.NewString()
	resp := getUserResponse(username, username+"123", []string{"abc", "bcd"})
	u := processResponseUnprovisioned(ctx, t, resp.Assertion, username, username+"123", s)

	require.True(t, u.Remote)

	groups, err := getUserGroups(ctx, u.ID)
	require.NoError(t, err)
	require.Contains(t, groups, "abc")
	require.Contains(t, groups, "bcd")
	require.Equal(t, len(groups), 3)

	_, err = user.StartSession(ctx, u)
	require.NoError(t, err)

	// test Update User fields based on SAML response
	resp = getUserResponse(username, username+"456", []string{"abc"})
	u = processResponseProvisioned(ctx, t, resp.Assertion, username, username+"456", s)

	require.True(t, u.Remote)

	groups2, err := getUserGroups(ctx, u.ID)
	require.NoError(t, err)
	require.Contains(t, groups2, "abc")
	require.NotContains(t, groups2, "bcd")
	require.Equal(t, len(groups2), 2)

	_, err = user.StartSession(ctx, u)
	require.NoError(t, err)
}

func TestSAMLWorkflowUserNotProvisioned(t *testing.T) {
	// First, make sure the mock SAML service is created.
	s := mockService(false)
	require.NotNil(t, s)

	ctx := context.Background()

	username := uuid.NewString()
	resp := getUserResponse(username, username+"123", []string{"abc", "bcd"})

	userAttr := s.toUserAttributes(resp.Assertion)
	require.Equal(t, username, userAttr.userName)

	_, err := user.ByUsername(ctx, userAttr.userName)
	log.Print(err)
	require.ErrorContains(t, err, "not found")
}

func TestSAMLWorkflowUserProvisioned(t *testing.T) {
	// First, make sure the mock SAML service is created.
	s := mockService(true)
	require.NotNil(t, s)

	ctx := context.Background()

	username := uuid.NewString()

	initialUser := &model.User{
		Username: username,
		Active:   true,
	}
	_, err := user.Add(ctx, initialUser, nil)
	require.NoError(t, err)

	resp := getUserResponse(username, username+"123", []string{"abc", "bcd"})
	u := processResponseProvisioned(ctx, t, resp.Assertion, username, username+"123", s)

	require.False(t, u.Remote)

	groups, err := getUserGroups(ctx, u.ID)
	require.NoError(t, err)
	require.Contains(t, groups, "abc")
	require.Contains(t, groups, "bcd")
	require.Equal(t, len(groups), 3)

	_, err = user.StartSession(ctx, u)
	require.NoError(t, err)
}

func mockService(autoProvision bool) *Service {
	service := &Service{
		db:           db.SingleDB(),
		samlProvider: nil,
		userConfig: userConfig{
			autoProvisionUsers:       autoProvision,
			groupsAttributeName:      "groups",
			displayNameAttributeName: "disp_name",
		},
	}
	return service
}

func getUserResponse(username string, dispName string, groups []string) saml.Response {
	resp := saml.Response{
		XMLName:      xml.Name{},
		IssueInstant: time.Time{},
		Status:       saml.Status{},
		Assertion:    &saml.Assertion{},
	}
	addAttribute(resp, "userName", username)
	addAttribute(resp, "disp_name", dispName)

	for _, g := range groups {
		addAttribute(resp, "groups", g)
	}

	return resp
}

func addAttribute(response saml.Response, name, value string) {
	if len(response.Assertion.AttributeStatements) == 0 {
		response.Assertion.AttributeStatements = append(response.Assertion.AttributeStatements, saml.AttributeStatement{})
	}
	response.Assertion.AttributeStatements[0].Attributes = append(response.Assertion.AttributeStatements[0].Attributes,
		saml.Attribute{
			FriendlyName: "",
			Name:         name,
			NameFormat:   "",
			Values: []saml.AttributeValue{
				{
					Type:  "xs:string",
					Value: value,
				},
			},
		})
}

func processResponseUnprovisioned(ctx context.Context, t *testing.T,
	response *saml.Assertion, username string, dispName string, s *Service,
) *model.User {
	userAttr := s.toUserAttributes(response)
	require.Equal(t, username, userAttr.userName)

	_, err := user.ByUsername(ctx, userAttr.userName)
	log.Print(err)
	require.True(t, errors.Is(err, db.ErrNotFound), true)
	u, err := s.provisionUser(ctx, userAttr.userName, userAttr.groups)
	require.NoError(t, err)

	u, err = s.syncUser(ctx, u, userAttr)
	require.NoError(t, err)

	require.Equal(t, dispName, u.DisplayName.String)
	require.Equal(t, username, u.Username)

	require.True(t, u.Active)

	return u
}

func processResponseProvisioned(ctx context.Context, t *testing.T,
	response *saml.Assertion, username string, dispName string, s *Service,
) *model.User {
	userAttr := s.toUserAttributes(response)
	require.Equal(t, username, userAttr.userName)

	u, err := user.ByUsername(ctx, userAttr.userName)
	require.NoError(t, err)

	u, err = s.syncUser(ctx, u, userAttr)
	require.NoError(t, err)

	require.Equal(t, dispName, u.DisplayName.String)
	require.Equal(t, username, u.Username)

	require.True(t, u.Active)

	return u
}

func getUserGroups(ctx context.Context, uID model.UserID) ([]string, error) {
	groups := []string{}
	err := db.Bun().NewSelect().TableExpr("user_group_membership AS ug").ColumnExpr("g.group_name").
		Where("ug.user_id = ?", uID).Join("LEFT OUTER JOIN groups g ON g.id = ug.group_id").Scan(ctx, &groups)
	return groups, err
}
