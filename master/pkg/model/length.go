package model

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/pkg/errors"
)

// Kind defines a kind of unit for specifying lengths.
type Kind string

// All the kinds available for Lengths.
const (
	Records Kind = "records"
	Batches Kind = "batches"
	Epochs  Kind = "epoches"
)

// Length a training duration in terms of records, batches or epochs.
type Length struct {
	Kind  Kind
	Units int
}

// MarshalJSON implements the json.Marshaler interface.
func (l Length) MarshalJSON() ([]byte, error) {
	switch l.Kind {
	case Records:
		return json.Marshal(map[string]int{
			"records": l.Units,
		})
	case Batches:
		return json.Marshal(map[string]int{
			"batches": l.Units,
		})
	case Epochs:
		return json.Marshal(map[string]int{
			"epochs": l.Units,
		})
	default:
		return json.Marshal(map[string]int{
			"batches": 0,
		})
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *Length) UnmarshalJSON(b []byte) error {
	var v map[string]int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	records, rOk := v["records"]
	batches, bOk := v["batches"]
	epochs, eOk := v["epochs"]

	switch {
	case rOk && !bOk && !eOk:
		*l = NewLengthInRecords(records)
	case !rOk && bOk && !eOk:
		*l = NewLengthInBatches(batches)
	case !rOk && !bOk && eOk:
		*l = NewLengthInEpochs(epochs)
	default:
		return errors.New(fmt.Sprintf("invalid length: %s", b))
	}

	return nil
}

// NewLength returns a new length with the specified unit and length.
func NewLength(kind Kind, units int) Length {
	return Length{Kind: kind, Units: units}
}

// NewLengthInRecords returns a new length in terms of records.
func NewLengthInRecords(records int) Length {
	return Length{Kind: Records, Units: records}
}

// NewLengthInBatches returns a new length in terms of batches.
func NewLengthInBatches(batches int) Length {
	return Length{Kind: Batches, Units: batches}
}

// NewLengthInEpochs returns a new Length in terms of epochs.
func NewLengthInEpochs(epochs int) Length {
	return Length{Kind: Epochs, Units: epochs}
}

func (l Length) String() string {
	return fmt.Sprintf("%d %s", l.Units, l.Kind)
}

// Validate implements the check.Validatable interface.
func (l Length) Validate() []error {
	return []error{}
}

// Add adds a length to another length.
func (l Length) Add(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units+other.Units)
}

// Sub subtracts a length from another length.
func (l Length) Sub(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units-other.Units)
}

// Mult multiplies a length by another length.
func (l Length) Mult(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units*other.Units)
}

// MultInt multiplies a length by an int.
func (l Length) MultInt(other int) Length {
	return NewLength(l.Kind, l.Units*other)
}

// Div divides a length by another length.
func (l Length) Div(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units/other.Units)
}

// DivInt divides a length by an int.
func (l Length) DivInt(other int) Length {
	return NewLength(l.Kind, l.Units/other)
}
