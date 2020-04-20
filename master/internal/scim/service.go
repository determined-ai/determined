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
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	scimPathRoot    = "/scim/v2"
	scimContentType = "application/scim+json"
)

type service struct {
	*Config
	db           *db.PgDB
	locationRoot *url.URL
}

func (s *service) validateSCIMCredentials(username, password string, c echo.Context) (bool, error) {
	if username == s.Config.Username && password == s.Config.Password {
		return true, nil
	}
	return false, nil
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

	users, err := s.db.SCIMUserList(startIndex, count, username)
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

	user, err := s.db.SCIMUserByID(id)
	if err != nil {
		return nil, err
	}

	if err := user.SetSCIMFields(s.locationRoot); err != nil {
		return nil, err
	}

	return user, nil
}

// PostUser creates a new SCIM user.
func (s *service) PostUser(c echo.Context) (interface{}, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	var user model.SCIMUser
	if err = json.Unmarshal(body, &user); err != nil {
		return nil, newBadRequestError(err)
	}

	if err = check.Validate(user); err != nil {
		return nil, newBadRequestError(err)
	} else if user.ID.Valid {
		return nil, newBadRequestError(errors.New("ID set"))
	}

	if user.Password.Valid {
		user.Password = null.StringFrom(replicateClientSideSaltAndHash(user.Password.String))
	}

	added, err := s.db.AddSCIMUser(&user)
	if err == db.ErrDuplicateRecord {
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

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	var user model.SCIMUser
	if err = json.Unmarshal(body, &user); err != nil {
		return nil, newBadRequestError(err)
	}

	if err = check.Validate(user); err != nil {
		return nil, newBadRequestError(err)
	} else if user.ID.String() != req.ID {
		return nil, newBadRequestError(errors.New("ID does not match path"))
	}

	updated, err := s.db.SetSCIMUser(req.ID, &user)
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

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, newBadRequestError(err)
	}

	if err = json.Unmarshal(body, &req.Patch); err != nil {
		return nil, newBadRequestError(err)
	}

	type Update struct {
		Active json.RawMessage `json:"active"`
		Emails json.RawMessage `json:"emails"`
		Name   json.RawMessage `json:"name"`
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
	}

	id, err := model.ParseUUID(req.ID)
	if err != nil {
		return nil, newNotFoundError(err)
	}

	changes.ID = id

	if err = changes.ValidateChanges(); err != nil {
		return nil, newBadRequestError(err)
	}

	updated, err := s.db.UpdateSCIMUser(req.ID, &changes, toUpdate)
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

const clientSidePasswordSalt = "GubPEmmotfiK9TMD6Zdw" // #nosec G101

// replicateClientSideSaltAndHash replicates the password salt and hash done on the client side.
// We need this because we hash passwords on the client side, but when SCIM posts a user with
// a password to password sync, it doesn't - so when we try to log in later, we get a weird,
// unrecognizable sha512 hash from the frontend.
func replicateClientSideSaltAndHash(password string) string {
	sum := sha512.Sum512([]byte(clientSidePasswordSalt + password))
	return fmt.Sprintf("%x", sum)
}
