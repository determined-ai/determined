package container

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// ID is a unique ID assigned to the containers of tasks when started in the cluster.
type ID string

// NewID provides a unique ID for a container.
func NewID() ID {
	return ID(uuid.New().String())
}

// String implements the fmt.Stringer interface.
func (id ID) String() string {
	return string(id)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (id ID) MarshalText() (text []byte, err error) {
	return []byte(string(id)), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (id *ID) UnmarshalText(text []byte) error {
	*id = ID(string(text))
	return nil
}

// Scan implements the sql.Scanner interface.
func (id *ID) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		*id = ID(src)
	case []byte:
		*id = ID(string(src))
	case nil:
		return nil
	default:
		return fmt.Errorf("incompatible type for container id: %T", src)
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}
