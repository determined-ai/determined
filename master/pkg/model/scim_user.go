package model

import (
	"crypto/sha512"
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
	ID         UUID       `db:"id" json:"id"`
	Username   string     `db:"username" json:"userName"`
	ExternalID string     `db:"external_id" json:"externalId"`
	Name       SCIMName   `db:"name" json:"name"`
	Emails     SCIMEmails `db:"emails" json:"emails"`
	Active     bool       `db:"active" json:"active"`

	PasswordHash null.String `db:"password_hash" json:"password_hash,omitempty"`

	Password string          `json:"password,omitempty"`
	Schemas  SCIMUserSchemas `json:"schemas"`
	Meta     *SCIMUserMeta   `json:"meta"`
}

// Validate checks that external data satisfies the expected invariants.
func (u SCIMUser) Validate() []error {
	var errs []error
	if len(u.Username) == 0 {
		errs = append(errs, errors.New("missing userName"))
	}

	return errs
}

// Sanitize sanitizes the user of external data that could be provided, but should
// always be ignored. See https://tools.ietf.org/html/rfc7643#section-3.1 for why
// meta must be cleared.
func (u *SCIMUser) Sanitize() {
	u.Meta = nil
}

// ValidateChanges checks that a patch for a user satisifies the expected
// invariants.
func (u SCIMUser) ValidateChanges() error {
	if !u.ID.Valid {
		return errors.New("missing ID")
	}

	return nil
}

// SetSCIMFields sets the location field for a user given the URL of the master
// and makes other changes, such as removing password fields from the model.
func (u *SCIMUser) SetSCIMFields(serverRoot *url.URL) error {
	l := *serverRoot
	l.Path = path.Join(l.Path, u.ID.String())

	u.Meta = &SCIMUserMeta{
		Location: l.String(),
	}

	u.Password = ""
	u.PasswordHash = EmptyPassword

	return nil
}

// UpdatePasswordHash updates the SCIMUser's password hash.
func (u *SCIMUser) UpdatePasswordHash(password string) error {
	if password == "" {
		u.PasswordHash = NoPasswordLogin
	} else {
		passwordHash := replicateClientSideSaltAndHash(password)
		passwordHash, err := HashPassword(passwordHash)
		if err != nil {
			return errors.Wrap(err, "error updating user password")
		}

		u.PasswordHash = null.StringFrom(passwordHash)
	}
	return nil
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
