package model

import (
	"database/sql/driver"
	"encoding/json"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"
)

// SCIMName is a name in SCIM.
type SCIMName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

// Value implements sql.Valuer.
func (e SCIMName) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan implements sql.Scanner.
func (e *SCIMName) Scan(value interface{}) error {
	return scanJSON(value, e)
}

// SCIMEmail is an email address in SCIM.
type SCIMEmail struct {
	Type    string `json:"type"`
	SValue  string `json:"value"`
	Primary bool   `json:"primary"`
}

// Value implements sql.Valuer.
func (e SCIMEmail) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan implements sql.Scanner.
func (e *SCIMEmail) Scan(value interface{}) error {
	return scanJSON(value, e)
}

// SCIMEmails is a list of emails in SCIM.
type SCIMEmails []SCIMEmail

// Value implements sql.Valuer.
func (e SCIMEmails) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan implements sql.Scanner.
func (e *SCIMEmails) Scan(value interface{}) error {
	return scanJSON(value, e)
}

// SCIMUserResourceType is the constant resource type field for users.
type SCIMUserResourceType struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMUserResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(scimUserType)
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMUserResourceType) UnmarshalJSON(data []byte) error {
	return validateString(scimUserType, data)
}

// SCIMUserMeta is the metadata for a user in SCIM.
type SCIMUserMeta struct {
	ResourceType SCIMUserResourceType `json:"resourceType"`
	Location     string               `json:"location"`
}

// SCIMUserSchemas is the constant schemas field for a user.
type SCIMUserSchemas struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMUserSchemas) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{scimUserSchema})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMUserSchemas) UnmarshalJSON(data []byte) error {
	return validateSchemas(scimUserSchema, data)
}

// SCIMUser is a user in SCIM.
type SCIMUser struct {
	ID         UUID        `db:"id" json:"id"`
	Username   string      `db:"username" json:"userName"`
	ExternalID string      `db:"external_id" json:"externalId"`
	Name       SCIMName    `db:"name" json:"name"`
	Emails     SCIMEmails  `db:"emails" json:"emails"`
	Active     bool        `db:"active" json:"active"`
	Password   null.String `json:"password,omitempty"`

	Schemas SCIMUserSchemas `json:"schemas"`
	Meta    *SCIMUserMeta   `json:"meta"`
}

// Validate checks that external data satisfies the expected invariants.
func (u SCIMUser) Validate() []error {
	var errs []error
	if u.Meta != nil {
		errs = append(errs, errors.New("meta set"))
	}
	if len(u.Username) == 0 {
		errs = append(errs, errors.New("missing userName"))
	}

	return errs
}

// ValidateChanges checks that a patch for a user satisifies the expected
// invariants.
func (u SCIMUser) ValidateChanges() error {
	if u.Meta != nil {
		return errors.New("meta set")
	}
	if !u.ID.Valid {
		return errors.New("missing ID")
	}

	return nil
}

// SetSCIMFields sets the location field for a user given the URL of the master.
func (u *SCIMUser) SetSCIMFields(serverRoot *url.URL) error {
	l := *serverRoot
	l.Path = path.Join(l.Path, u.ID.String())

	u.Meta = &SCIMUserMeta{
		Location: l.String(),
	}

	return nil
}

// SCIMUsers is a list of users in SCIM.
type SCIMUsers struct {
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	Resources    []*SCIMUser `json:"Resources"`

	ItemsPerPage int             `json:"itemsPerPage"`
	Schemas      SCIMListSchemas `json:"schemas"`
}

// SetSCIMFields sets the location field for all users given the URL of the
// master.
func (u *SCIMUsers) SetSCIMFields(serverRoot *url.URL) error {
	for _, u := range u.Resources {
		if err := u.SetSCIMFields(serverRoot); err != nil {
			return err
		}
	}

	u.ItemsPerPage = len(u.Resources)

	return nil
}
