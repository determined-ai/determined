package oidc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	cookieName          = "oauth2_state"
	cookieTTL           = 5 * 60
	defaultRedirectPath = "/det/login"
	// This must match the value at $PROJECT_ROOT/cli/determined_cli/sso.CLI_REDIRECT_PORT.
	cliRedirectPath         = "http://localhost:49176"
	deprecatedCliRelayState = "cli=true"
	cliRelayState           = "cli"
	envVarName              = "DET_OIDC_CLIENT_SECRET"
)

// Service handles OIDC interactions.
type Service struct {
	config       config.OIDCConfig
	db           *db.PgDB
	provider     *oidc.Provider
	oauth2Config oauth2.Config
}

// IDTokenClaims represents the set of claims in an OIDC ID token that we're concerned with.
type IDTokenClaims struct {
	AuthenticationClaim string   `json:"authentication_claim"`
	DisplayName         string   `json:"display_name"`
	Groups              []string `json:"groups"`
}

var errNotProvisioned = echo.NewHTTPError(http.StatusNotFound, "user has not been provisioned")

// New initiates an OIDC Service.
func New(db *db.PgDB, config config.OIDCConfig) (*Service, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, config.IDPSSOURL)
	if err != nil {
		return nil, err
	}

	ru, err := url.Parse(config.IDPRecipientURL)
	if err != nil {
		return nil, err
	}
	// join instead of replacing path in case we're behind a rewriting proxy
	ru.Path = path.Join(ru.Path, OidcRoot, CallbackPath)

	secret := config.ClientSecret
	if secret == "" {
		secret = os.Getenv(envVarName)
	}
	if secret == "" {
		return nil, fmt.Errorf("client secret has not been set")
	}

	return &Service{
		config:   config,
		db:       db,
		provider: provider,
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: secret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  ru.String(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}, nil
}

// callback validates the response from the OIDC provider, checking that the
// request matches the response, the oauth2 token is valid, and that the user
// is active.
func (s *Service) callback(c echo.Context) error {
	oauth2token, err := s.getOauthToken(c)
	if err != nil {
		return errors.Wrap(err, "failed to exchange oauth2 token")
	}

	rawIDToken, ok := oauth2token.Extra("id_token").(string)
	if !ok {
		return errors.Wrap(err, "failed to get raw ID token from oauth2token")
	}
	userInfo, err := s.provider.UserInfo(c.Request().Context(), oauth2.StaticTokenSource(oauth2token))
	if err != nil {
		return errors.Wrap(err, "failed to get user info from oidc provider")
	}

	claims, err := s.toIDTokenClaim(userInfo)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	u, err := s.lookupUser(ctx, claims.AuthenticationClaim)
	if errors.Is(err, db.ErrNotFound) && s.config.AutoProvisionUsers {
		newUser, err := s.provisionUser(ctx, claims.AuthenticationClaim, claims.Groups)
		if err != nil {
			return err
		}
		u = newUser
	} else if err != nil {
		return err
	}

	if !u.Remote {
		return echo.NewHTTPError(http.StatusBadRequest,
			"user exists but was not created with the --remote option")
	}

	u, err = s.syncUser(ctx, u, claims)
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"auth-claim":       s.config.AuthenticationClaim,
		"scim-attribute":   s.config.SCIMAuthenticationAttribute,
		"auth-claim-value": claims.AuthenticationClaim,
	}).Info("provisioned & synced user given claims")

	if !u.Active {
		return echo.NewHTTPError(http.StatusBadRequest, "user is inactive")
	}
	token, err := user.StartSession(ctx, u, user.WithInheritedClaims(map[string]string{"OIDCRawIDToken": rawIDToken}))
	if err != nil {
		return err
	}

	c.SetCookie(user.NewCookieFromToken(token))
	redirectPath := defaultRedirectPath
	switch relayState := c.QueryParam("relayState"); relayState {
	case deprecatedCliRelayState:
		fallthrough
	case cliRelayState:
		redirectPath = cliRedirectPath + fmt.Sprintf("?token=%s", url.QueryEscape(token))
	case "":
		// do nothing to the default redirectPath
	default:
		redirectPath += fmt.Sprintf("?relayState=%s", url.QueryEscape(relayState))
	}

	return c.Redirect(http.StatusSeeOther, redirectPath)
}

// getOauthToken returns the oauth2.Token from the oidc cookie.
func (s *Service) getOauthToken(c echo.Context) (*oauth2.Token, error) {
	state, err := c.Cookie(cookieName)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve state cookie")
	}
	if c.QueryParam("state") != state.Value {
		return nil, errors.New("oidc state did not match")
	}

	relayParam := c.QueryParam("relayState")
	// Tolerate older CLI versions (<=0.19.5)
	if relayParam == cliRelayState || relayParam == deprecatedCliRelayState {
		configCopy := s.oauth2Config
		configCopy.RedirectURL = fmt.Sprintf("%s?relayState=%s", configCopy.RedirectURL, relayParam)
		return configCopy.Exchange(c.Request().Context(), c.QueryParam("code"))
	}
	return s.oauth2Config.Exchange(c.Request().Context(), c.QueryParam("code"))
}

// toIDTokenClaim takes the user info & parses out the claims into an IDTokenClaim struct.
func (s *Service) toIDTokenClaim(userInfo *oidc.UserInfo) (*IDTokenClaims, error) {
	var cs map[string]interface{}
	if err := userInfo.Claims(&cs); err != nil {
		return nil, errors.Wrap(err, "failed to extract OIDC claims")
	}

	c := IDTokenClaims{}

	if cs[s.config.AuthenticationClaim] != nil {
		authValue, ok := cs[s.config.AuthenticationClaim].(string)
		if !ok {
			return nil, fmt.Errorf("user info authenticationClaim value was not a string")
		}
		c.AuthenticationClaim = authValue
	} else {
		return nil, fmt.Errorf("user info authenticationClaim missing")
	}

	if cs["display_name"] != nil {
		displayName, ok := cs["display_name"].(string)
		if !ok {
			return nil, fmt.Errorf("user info displayName value was not a string")
		}
		c.DisplayName = displayName
	}

	if cs[s.config.GroupsClaimName] != nil {
		gs, ok := cs[s.config.GroupsClaimName].([]interface{})
		if !ok {
			return nil, fmt.Errorf("user info groups value was not a slice")
		}

		groups := make([]string, len(gs))
		for i, val := range gs {
			v, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("user info value was not a string: %s", val)
			}
			groups[i] = v
		}

		c.Groups = groups
	}
	return &c, nil
}

// lookupUser: First try finding user in our users.scim table.
// If we don't find them there and the scim attribute is userName & look in the user table.
func (s *Service) lookupUser(ctx context.Context, claimValue string) (*model.User, error) {
	u, err := s.db.UserBySCIMAttribute(s.config.SCIMAuthenticationAttribute, claimValue)
	if errors.Is(err, db.ErrNotFound) {
		if s.config.SCIMAuthenticationAttribute != "userName" {
			return nil, errNotProvisioned
		}
		return user.ByUsername(ctx, claimValue)
	} else if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return u, err
}

// syncUser syncs the mutable user fields parsed from the claim, only if there are non-null changes.
func (s *Service) syncUser(ctx context.Context, u *model.User, claims *IDTokenClaims) (*model.User, error) {
	if err := db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable},
		func(ctx context.Context, tx bun.Tx) error {
			// If the config is set to auto-provision users, sync the display name.
			if s.config.AutoProvisionUsers {
				if claims.DisplayName != "" && claims.DisplayName != u.DisplayName.String {
					if _, err := tx.NewUpdate().
						Model(&model.User{
							ID:          u.ID,
							Username:    claims.AuthenticationClaim,
							DisplayName: null.NewString(claims.DisplayName, true),
						}).Column("display_name").Where("id = ?", u.ID).Exec(ctx); err != nil {
						return fmt.Errorf("error setting display name of %q: %s", u.Username, err)
					}
				}
			}
			if s.config.GroupsClaimName != "" {
				if err := s.updateUserGroupMembership(ctx, tx, u, claims.Groups); err != nil {
					return fmt.Errorf("could not update user group membership: %s", err)
				}
			}
			return nil
		}); err != nil {
		return nil, err
	}
	return user.ByUsername(ctx, u.Username)
}

// provisionUser: If we get forwarded an ID token for an unknown user from the IdP,
// create a remote user with no password in the user table.
func (s *Service) provisionUser(
	ctx context.Context,
	username string,
	groups []string,
) (*model.User, error) {
	u := model.User{
		Username:     username,
		PasswordHash: model.NoPasswordLogin,
		Active:       true,
		Remote:       true,
	}

	if err := db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable},
		func(ctx context.Context, tx bun.Tx) error {
			if _, err := user.AddUserTx(ctx, tx, &u); err != nil {
				return errNotProvisioned
			}
			if s.config.GroupsClaimName != "" {
				if err := s.updateUserGroupMembership(ctx, tx, &u, groups); err != nil {
					return fmt.Errorf("could not update user group membership: %s", err)
				}
			}
			return nil
		}); err != nil {
		return nil, err
	}
	return user.ByUsername(ctx, username)
}

func (s *Service) updateUserGroupMembership(ctx context.Context, tx bun.IDB, u *model.User, groups []string) error {
	// Get a list of groups a user is in.
	currentGroups, err := usergroup.SearchGroupsWithoutPersonalGroupsTx(ctx, tx, "", u.ID)
	if err != nil {
		return fmt.Errorf("finding current user groups: %w", err)
	}

	var groupsToRemove []int
	// Remove the user from any groups no longer included in the claim.
	for _, g := range currentGroups {
		if !slices.Contains(groups, g.Name) {
			groupsToRemove = append(groupsToRemove, g.ID)
		}
	}
	if len(groupsToRemove) != 0 {
		if err := usergroup.RemoveUsersFromGroupsTx(ctx, tx, groupsToRemove, u.ID); err != nil {
			return fmt.Errorf("failed to remove user from group: %w", err)
		}
	}

	var groupsToAdd []int
	// Add the user to groups included in the claim.
	for _, g := range groups {
		// Check if the group already exists, regardless of if the user belongs to it.
		gps, err := usergroup.SearchGroupsWithoutPersonalGroupsTx(ctx, tx, g, model.UserID(0))
		if err != nil {
			return fmt.Errorf("failed to find usergroup: %w", err)
		}
		if len(gps) == 0 {
			continue // TODO DET-9937
		}
		// If the group exists in the system but isn't part of the user's registered groups, update.
		// gps should be a slice of length 1 since group name is unique.
		if !slices.Contains(currentGroups, gps[0]) {
			groupsToAdd = append(groupsToAdd, gps[0].ID)
		}
	}
	if len(groupsToAdd) != 0 {
		if err := usergroup.AddUsersToGroupsTx(ctx, tx, groupsToAdd, true, u.ID); err != nil {
			return fmt.Errorf("error adding user to group: %s", err)
		}
	}

	return nil
}

// initiate saves a random string as a cookie and redirects the user to the
// configured OIDC provider. The OIDC provider return the random string in
// order to establish request/response correlation.
func (s *Service) initiate(c echo.Context) error {
	state, err := randString(16)
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    state,
		MaxAge:   cookieTTL,
		Secure:   true,
		HttpOnly: true,
	})

	relayState := map[string]string{"relayState": c.QueryParam("relayState")}
	return c.Redirect(http.StatusFound, authCodeURLWithParams(s.oauth2Config, state, relayState))
}

// randString generates n randomized chars.
func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// authCodeURLWithParams attaches the specified key:value pairs as querystring
// parameters to the redirect URL used by AuthCodeURL.
func authCodeURLWithParams(conf oauth2.Config, state string, kv map[string]string) string {
	u, err := url.Parse(conf.RedirectURL)
	if err != nil {
		return conf.AuthCodeURL(state)
	}
	queries := u.Query()
	for k, v := range kv {
		if v != "" {
			queries.Add(k, v)
		}
	}

	u.RawQuery = queries.Encode()
	conf.RedirectURL = u.String()
	return conf.AuthCodeURL(state)
}
