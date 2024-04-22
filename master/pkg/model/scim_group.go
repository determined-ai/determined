package model

import (
	"encoding/json"
	"net/url"
)

// SCIMGroupResourceType is the constant resource type field for groups.
type SCIMGroupResourceType struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMGroupResourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(scimGroupType)
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMGroupResourceType) UnmarshalJSON(data []byte) error {
	return validateString(scimGroupType, data)
}

// SCIMGroupMeta is the metadata for a group in SCIM.
type SCIMGroupMeta struct {
	ResourceType SCIMGroupResourceType `json:"resourceType"`
}

// SCIMGroupSchemas is the constant schemas field for a user.
type SCIMGroupSchemas struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMGroupSchemas) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{scimGroupSchema})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMGroupSchemas) UnmarshalJSON(data []byte) error {
	return validateSchemas(scimGroupSchema, data)
}

// SCIMGroup is a group in SCIM.
type SCIMGroup struct {
	ID          UUID        `json:"id"`
	DisplayName string      `json:"displayName"`
	Members     []*SCIMUser `json:"members"`

	Schemas SCIMGroupSchemas `json:"schemas"`
	Meta    *SCIMGroupMeta   `json:"meta"`
}

// SCIMGroups is a list of groups in SCIM.
type SCIMGroups struct {
	TotalResults int          `json:"totalResults"`
	StartIndex   int          `json:"startIndex"`
	Resources    []*SCIMGroup `json:"Resources"`

	ItemsPerPage int             `json:"itemsPerPage"`
	Schemas      SCIMListSchemas `json:"schemas"`
}

// SetSCIMFields sets the location field for all users given the URL of the
// master.
func (g *SCIMGroups) SetSCIMFields(serverRoot *url.URL) error {
	g.ItemsPerPage = len(g.Resources)

	return nil
}
