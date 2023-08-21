package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/check"
)

// Unit is the type of unit for specifying lengths.
type Unit string

// All the units available for lengths.
const (
	Records     Unit = "records"
	Batches     Unit = "batches"
	Epochs      Unit = "epochs"
	Unitless    Unit = "unitless"
	Unspecified Unit = "unspecified"
)

// LengthV0 a training duration in terms of records, batches or epochs.
type LengthV0 struct {
	Unit  Unit
	Units uint64
}

// MarshalJSON implements the json.Marshaler interface.
func (l LengthV0) MarshalJSON() ([]byte, error) {
	switch l.Unit {
	case Records:
		return json.Marshal(map[string]uint64{
			"records": l.Units,
		})
	case Batches:
		return json.Marshal(map[string]uint64{
			"batches": l.Units,
		})
	case Epochs:
		return json.Marshal(map[string]uint64{
			"epochs": l.Units,
		})
	case Unitless:
		return json.Marshal(l.Units)
	default:
		return json.Marshal(map[string]uint64{
			"batches": 0,
		})
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *LengthV0) UnmarshalJSON(b []byte) error {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		// Just a plain integer means a Unitless Length.
		*l = NewLengthUnitless(n)
		return nil
	}

	var v map[string]uint64
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
		return fmt.Errorf("invalid length: %s", b)
	}

	return nil
}

// NewLength returns a new length with the specified unit and length.
func NewLength(unit Unit, units uint64) LengthV0 {
	return LengthV0{Unit: unit, Units: units}
}

// NewLengthInRecords returns a new length in terms of records.
func NewLengthInRecords(records uint64) LengthV0 {
	return LengthV0{Unit: Records, Units: records}
}

// NewLengthInBatches returns a new length in terms of batches.
func NewLengthInBatches(batches uint64) LengthV0 {
	return LengthV0{Unit: Batches, Units: batches}
}

// NewLengthInEpochs returns a new LengthV0 in terms of epochs.
func NewLengthInEpochs(epochs uint64) LengthV0 {
	return LengthV0{Unit: Epochs, Units: epochs}
}

// NewLengthUnitless returns a new LengthV0 with no assigned units.
func NewLengthUnitless(n uint64) LengthV0 {
	return LengthV0{Unit: Unitless, Units: n}
}

func (l LengthV0) String() string {
	return fmt.Sprintf("%d %s", l.Units, l.Unit)
}

// Add adds a length to another length.
func (l LengthV0) Add(other LengthV0) LengthV0 {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units+other.Units)
}

// Sub subtracts a length from another length.
func (l LengthV0) Sub(other LengthV0) LengthV0 {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units-other.Units)
}

// Mult multiplies a length by another length.
func (l LengthV0) Mult(other LengthV0) LengthV0 {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units*other.Units)
}

// MultInt multiplies a length by an int.
func (l LengthV0) MultInt(other uint64) LengthV0 {
	return NewLength(l.Unit, l.Units*other)
}

// Div divides a length by another length.
func (l LengthV0) Div(other LengthV0) LengthV0 {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units/other.Units)
}

// DivInt divides a length by an int.
func (l LengthV0) DivInt(other uint64) LengthV0 {
	return NewLength(l.Unit, l.Units/other)
}
