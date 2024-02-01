//go:build integration
// +build integration

package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
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

func TestOIDCWorkflow(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name            string
		userName        string
		groups          []string
		createDuplicate bool
	}{
		{"user-provisioned", uuid.NewString(), []string{"abc"}, false},
		{"user-exists", uuid.NewString(), []string{"abc"}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svr := newHTTPTestServer(t, tt.userName+"@hpe.com", "1234567", tt.userName, tt.groups)
			defer svr.Close()

			// Add a fake user to the db with the same email but DIFFERENT display name.
			if tt.createDuplicate {
				createAndAddFakeUser(ctx, t, tt.userName)
			}

			// First, make sure the mock OIDC service is created.
			s := mockService(t, svr.URL)
			require.NotNil(t, s)

			// Then, test the different OIDC functions, with the mock server & OIDC service.
			tok, err := s.getOauthToken(mockContext(svr.URL))
			require.NoError(t, err)
			require.NotNil(t, tok)

			info, err := s.provider.UserInfo(context.Background(), oauth2.StaticTokenSource(tok))
			require.NotNil(t, info)
			require.NoError(t, err)

			claims, err := s.toIDTokenClaim(info)
			require.NoError(t, err)
			require.NotNil(t, claims)

			u, err := s.lookupUser(ctx, claims.AuthenticationClaim)
			if tt.createDuplicate {
				require.NoError(t, err)
			} else {
				require.Error(t, err, db.ErrNotFound)
				newUser, err := s.provisionUser(ctx, claims.AuthenticationClaim, claims.Groups)
				require.NoError(t, err)
				u = newUser
			}

			require.True(t, u.Remote)

			u, err = s.syncUser(ctx, u, claims)
			require.NoError(t, err)

			require.True(t, u.Active)

			// Now check that all user fields match the response.
			require.Equal(t, tt.userName, u.DisplayName.String)
			require.Equal(t, tt.userName+"@hpe.com", u.Username)

			actualGroups := []string{}
			err = db.Bun().NewSelect().TableExpr("user_group_membership AS ug").ColumnExpr("g.group_name").
				Where("ug.user_id = ?", u.ID).Join("LEFT OUTER JOIN groups g ON g.id = ug.group_id").Scan(ctx, &actualGroups)
			require.NoError(t, err)

			// Checking that tt.groups is a subset of actualGroups, which will contain the user's personal group.
			require.Subset(t, actualGroups, tt.groups)

			_, err = user.StartSession(ctx, u)
			require.NoError(t, err)
		})
	}
}

func TestFailOauthToken(t *testing.T) {
	svr := newHTTPTestServer(t, "", "", "", []string{})
	defer svr.Close()

	s := mockService(t, svr.URL)
	require.NotNil(t, s)

	tok, err := s.getOauthToken(mockContext(svr.URL))
	require.ErrorContains(t, err, "could not exchange auth token")
	require.Nil(t, tok)
}

func TestFailToExtractClaims(t *testing.T) {
	svr := newHTTPTestServer(t, "", "1234567", "", []string{})
	defer svr.Close()

	s := mockService(t, svr.URL)
	require.NotNil(t, s)

	tok, err := s.getOauthToken(mockContext(svr.URL))
	require.NoError(t, err)
	require.NotNil(t, tok)

	info, err := s.provider.UserInfo(context.Background(), oauth2.StaticTokenSource(tok))
	require.NotNil(t, info)
	require.NoError(t, err)

	// If the email is missing from the claim, fail.
	claims, err := s.toIDTokenClaim(info)
	require.ErrorContains(t, err, "user info authenticationClaim missing")
	require.Nil(t, claims)
}

func mockService(t *testing.T, url string) *Service {
	clientID := "123456"
	clientSecret := "abcdefgh"
	idpssoURL := fmt.Sprint(url)

	p, err := oidc.NewProvider(context.Background(), idpssoURL)
	require.NoError(t, err)

	return &Service{
		config: config.OIDCConfig{
			Enabled:                     true,
			Provider:                    "Okta",
			ClientID:                    clientID,
			ClientSecret:                clientSecret,
			IDPSSOURL:                   idpssoURL,
			IDPRecipientURL:             "https://dev-123456.okta.com",
			AuthenticationClaim:         "email",
			SCIMAuthenticationAttribute: "userName",
			AutoProvisionUsers:          true,           // Hard-coding these variables
			GroupsAttributeName:         "groups",       // for testing purposes,
			DisplayNameAttributeName:    "display_name", // but they can be customized.
		},
		provider: p,
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     p.Endpoint(),
			RedirectURL:  fmt.Sprint(url + "/oidc/callback"),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
		},
	}
}

func mockContext(url string) echo.Context {
	params := map[string]string{
		"state":      "SUCCESS",
		"relayState": "",
		"code":       "1234567",
	}

	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Value: "SUCCESS", Name: "oauth2_state"})

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	return echo.New().NewContext(req, httptest.NewRecorder())
}

func newHTTPTestServer(t *testing.T, email string, accessToken string,
	dispName string, groups []string,
) *httptest.Server {
	var svr *httptest.Server
	svr = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(echo.HeaderContentType, "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(writeResponse(t, svr.URL, email, accessToken, dispName, groups))
		require.NoError(t, err)
	}))

	return svr
}

func writeResponse(t *testing.T, url string, email string, accessToken string,
	dispName string, groups []string,
) []byte {
	groupResponse := map[string][]string{
		"groups": groups,
	}
	groupBytes, err := json.Marshal(groupResponse)
	require.NoError(t, err)

	strResponse := map[string]string{
		"issuer":                 url,
		"authorization_endpoint": url + "/authorize",
		"token_endpoint":         url + "/token",
		"userinfo_endpoint":      url + "/userinfo",
		"jwks_uri":               url + "/.well-known/jwks.json",
		"access_token":           accessToken,
		"authentication_claim":   "email",
	}

	if email != "" {
		strResponse["email"] = email
	}

	if dispName != "" {
		strResponse["display_name"] = dispName
	}

	b, err := json.Marshal(strResponse)
	require.NoError(t, err)

	fullResponse := [][]byte{b[:len(b)-1], groupBytes[1:]}
	return bytes.Join(fullResponse, []byte(", "))
}

func createAndAddFakeUser(ctx context.Context, t *testing.T, userName string) {
	fakeUser := model.User{
		Username:    userName + "@hpe.com",
		DisplayName: null.StringFrom("fake-username"),
		Active:      true,
		Remote:      true,
	}

	// If running this test multiple times, check if fake user is already in the DB.
	if _, err := user.ByUsername(ctx, fakeUser.Username); errors.Is(err, db.ErrNotFound) {
		uID, err := user.Add(context.Background(), &fakeUser, nil)
		require.NoError(t, err)
		fakeUser.ID = uID

		err = usergroup.UpdateUserGroupMembershipTx(ctx, db.Bun(), &fakeUser, []string{"bcd"})
		require.NoError(t, err)
	}
}
