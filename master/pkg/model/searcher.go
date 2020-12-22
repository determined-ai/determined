package model

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/google/uuid"
)

// Snapshotter is any object that implements how to save an restore its state.
type Snapshotter interface {
	Snapshot() (json.RawMessage, error)
	Restore(json.RawMessage) error
}

// RequestID links all operations with the same ID to a single trial create request.
type RequestID uuid.UUID

// NewRequestID returns a new request ID using the provided reader.
func NewRequestID(r io.Reader) RequestID {
	var u uuid.UUID
	if _, err := io.ReadFull(r, u[:]); err != nil {
		// We always read from an `nprand.State`, which should
		// not return an error in practice.
		panic(fmt.Sprintf("unexpected error creating request ID: %v", err))
	}

	// Ensure that the underlying UUID is a valid UUIDv4.
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4.
	u[8] = (u[8] & 0x3f) | 0x80 // Variant is 10.
	return RequestID(u)
}

// MarshalText returns the marshaled form of this ID, which is the string form of the underlying
// UUID.
func (r RequestID) MarshalText() ([]byte, error) {
	return []byte(uuid.UUID(r).String()), nil
}

// UnmarshalText unmarshals this ID from a text representation.
func (r *RequestID) UnmarshalText(data []byte) error {
	u, err := uuid.ParseBytes(data)
	if err != nil {
		return err
	}
	*r = RequestID(u)
	return nil
}

// Before determines whether this UUID is strictly lexicographically less (comparing the sequences
// of bytes) than another one.
func (r RequestID) Before(s RequestID) bool {
	return bytes.Compare(r[:], s[:]) == -1
}

func (r RequestID) String() string {
	return uuid.UUID(r).String()
}

// ParseRequestID decodes s into a request id or returns an error.
func ParseRequestID(s string) (RequestID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return RequestID{}, err
	}
	return RequestID(parsed), nil
}

// MustParseRequestID decodes s into a request id or panics.
func MustParseRequestID(s string) RequestID {
	parsed, err := ParseRequestID(s)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Value implements the sql.Driver interface.
func (r RequestID) Value() (driver.Value, error) {
	return r.String(), nil
}

// Scan implements the sql.Scanner interface.
func (r *RequestID) Scan(value interface{}) error {
	var x uuid.UUID
	var err error
	switch v := value.(type) {
	case string:
		x, err = uuid.Parse(v)
	case []byte:
		x, err = uuid.Parse(string(v))
	default:
		return errors.Errorf("unknown type %T", v)
	}

	if err != nil {
		return errors.Wrap(err, "failed to scan RequestID")
	}

	*r = RequestID(x)
	return nil
}
