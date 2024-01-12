// Package scim partially implements the SCIM v2.0 protocol as described by
// RFC7644 [1].  In general, the SCIM v2.0 protocol is a REST-ful API that
// identity providers (IdPs) use to insert users into Determined upon user
// provisioning in the IdP.
//
// For expediency, this package only implements the subset of the protocol used
// by the Okta IdP [2,3].
//
// [1] https://tools.ietf.org/html/rfc7644
// [2] https://developer.okta.com/docs/concepts/scim/
// [3] https://github.com/oktadeveloper/okta-scim-beta
package scim

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/plugin/oauth"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	scimPathRoot    = "/scim/v2"
	scimContentType = "application/scim+json"
)

type service struct {
	config       *config.ScimConfig
	db           *db.PgDB
	locationRoot *url.URL
	oauthService *oauth.Service
}

func (s *service) validateBasicAuth(username, password string, c echo.Context) (bool, error) {
	if username == "" || password == "" || s.config.Auth.BasicAuthConfig == nil {
		return false, nil
	}
	config := s.config.Auth.BasicAuthConfig
	return username == config.Username && password == config.Password, nil
}

func (s *service) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch {
		case s.config.Auth.BasicAuthConfig != nil:
			return middleware.BasicAuth(s.validateBasicAuth)(next)(c)

		case s.config.Auth.OAuthConfig != nil:
			if oauthValid, _ := s.oauthService.ValidateRequest(c); !oauthValid {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid OAuth credentials")
			}
			return next(c)
		}
		return echo.NewHTTPError(
			http.StatusInternalServerError, "no authentication method configured for SCIM",
		)
	}
}

// GetUsers returns a list of SCIM users, which may be optionally filtered.
func (s *service) GetUsers(c echo.Context) (interface{}, error) {
	type Request struct {
		Filter     *string `query:"filter"`
		Count      *int    `query:"count"`
		StartIndex *int    `query:"startIndex"`
	}

	var req Request
	if err := api.BindArgs(&req, c); err != nil {
		return nil, errors.WithStack(err)
	}

	count := 100
	if req.Count != nil {
		count = *req.Count
	}
	if count < 0 {
		return nil, newBadRequestError(errors.New("count < 0"))
	}

	startIndex := 0
	if req.StartIndex != nil {
		startIndex = *req.StartIndex
	}
	if startIndex < 0 {
		return nil, newBadRequestError(errors.New("startIndex < 0"))
	}

	// Okta will only filter on userName.
	var username string
	const q = "userName eq "
	if f := req.Filter; f != nil && len(*f) != 0 {
		if strings.HasPrefix(*f, q) {
			v, err := strconv.Unquote(strings.TrimPrefix(*f, q))
			if err != nil {
				return nil, newBadRequestError(err)
			}
			username = v
		} else {
			return nil, newBadRequestError(errors.New("unsupported filter"))
		}
	}

	users, err := user.SCIMUserList(c.Request().Context(), startIndex, count, username)
	if err != nil {
		return nil, err
	}

	if err := users.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	if users.Resources == nil {
		users.Resources = make([]*model.SCIMUser, 0)
	}

	return users, nil
}

// GetUser returns a SCIM user by ID.
func (s *service) GetUser(c echo.Context) (interface{}, error) {
	type Request struct {
		ID string `path:"user_id"`
	}

	var req Request
	if err := api.BindArgs(&req, c); err != nil {
		return nil, err
	}

	id, err := model.ParseUUID(req.ID)
	if err != nil {
		return nil, newNotFoundError(err)
	}

	u, err := user.SCIMUserByID(c.Request().Context(), db.Bun(), id)
	if err != nil {
		return nil, err
	}

	if err := u.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	return u, nil
}

// PostUser creates a new SCIM user.
func (s *service) PostUser(c echo.Context) (interface{}, error) {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	var u model.SCIMUser
	if err = json.Unmarshal(body, &u); err != nil {
		return nil, newBadRequestError(err)
	}
	if err = json.Unmarshal(body, &u.RawAttributes); err != nil {
		return nil, newBadRequestError(err)
	}

	if err = check.Validate(u); err != nil {
		return nil, newBadRequestError(err)
	} else if u.ID.Valid {
		return nil, newBadRequestError(errors.New("ID set"))
	}

	u.Sanitize()

	err = u.UpdatePasswordHash(u.Password)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	added, err := user.AddSCIMUser(c.Request().Context(), &u)
	if errors.Is(err, db.ErrDuplicateRecord) {
		return nil, newConflictError(err)
	} else if err != nil {
		return nil, err
	}

	if err = added.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	c.Response().Header().Set("Location", added.Meta.Location)
	c.Response().Status = http.StatusCreated

	return added, nil
}

// PutUser updates all the fields of an existing SCIM user.
func (s *service) PutUser(c echo.Context) (interface{}, error) {
	type Request struct {
		ID string `path:"user_id"`
	}

	var req Request
	if err := api.BindArgs(&req, c); err != nil {
		return nil, errors.WithStack(err)
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	var u model.SCIMUser
	if err = json.Unmarshal(body, &u); err != nil {
		return nil, newBadRequestError(err)
	}
	if err = json.Unmarshal(body, &u.RawAttributes); err != nil {
		return nil, newBadRequestError(err)
	}

	if err = check.Validate(u); err != nil {
		return nil, newBadRequestError(err)
	} else if u.ID.String() != req.ID {
		return nil, newBadRequestError(errors.New("ID does not match path"))
	}

	u.Sanitize()

	err = u.UpdatePasswordHash(u.Password)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	updated, err := user.SetSCIMUser(c.Request().Context(), req.ID, &u)
	if err != nil {
		return nil, err
	}

	if err := updated.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	return updated, nil
}

// PatchUser updates specific fields of an existing SCIM user. The format of the
// request is a JSON patch (RFC 6902).
func (s *service) PatchUser(c echo.Context) (interface{}, error) {
	updatedFields := make(map[string]bool)
	var toUpdate []string

	// parseField is a helper function that keeps track of unmarshalled fields.
	parseField := func(bs []byte, fieldName string, dst interface{}) error {
		if len(bs) == 0 {
			return nil
		}

		if updatedFields[fieldName] {
			return newBadRequestError(errors.Errorf("field %s already replaced", fieldName))
		}
		updatedFields[fieldName] = true

		if err := json.Unmarshal(bs, dst); err != nil {
			return newBadRequestError(err)
		}
		toUpdate = append(toUpdate, fieldName)

		return nil
	}

	type Request struct {
		ID    string `path:"user_id"`
		Patch model.PatchRequest
	}

	var req Request
	if err := api.BindArgs(&req, c); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	if err = json.Unmarshal(body, &req.Patch); err != nil {
		return nil, newBadRequestError(err)
	}

	type Update struct {
		Active      json.RawMessage `json:"active"`
		Emails      json.RawMessage `json:"emails"`
		Name        json.RawMessage `json:"name"`
		DisplayName json.RawMessage `json:"displayName"`
	}

	var changes model.SCIMUser
	for _, op := range req.Patch.Operations {
		if op.Op != "replace" {
			return nil, newBadRequestError(errors.New("only replace is supported"))
		}

		if len(op.Path) != 0 {
			return nil, newBadRequestError(errors.New("updating subpaths is not supported"))
		}

		dec := json.NewDecoder(bytes.NewReader(op.Value))
		dec.DisallowUnknownFields()

		var u Update
		if err = dec.Decode(&u); err != nil {
			return nil, newBadRequestError(err)
		}

		if err = parseField(u.Active, "active", &changes.Active); err != nil {
			return nil, err
		}

		if err = parseField(u.Emails, "emails", &changes.Emails); err != nil {
			return nil, err
		}

		if err = parseField(u.Name, "name", &changes.Name); err != nil {
			return nil, err
		}

		if err = parseField(u.DisplayName, "display_name", &changes.DisplayName); err != nil {
			return nil, err
		}
	}

	id, err := model.ParseUUID(req.ID)
	if err != nil {
		return nil, newNotFoundError(err)
	}

	changes.ID = id

	if err = changes.ValidateChanges(); err != nil {
		return nil, newBadRequestError(err)
	}

	updated, err := user.UpdateUserAndDeleteSession(c.Request().Context(), req.ID, &changes, toUpdate)
	if err != nil {
		return nil, err
	}

	if err := updated.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *service) GetGroups(c echo.Context) (interface{}, error) {
	var groups model.SCIMGroups

	if err := groups.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	if groups.Resources == nil {
		groups.Resources = make([]*model.SCIMGroup, 0)
	}

	return groups, nil
}
