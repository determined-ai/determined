package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

// Unit is the type of unit for specifying lengths.
type Unit string

// All the units available for lengths.
const (
	Records     Unit = "records"
	Batches     Unit = "batches"
	Epochs      Unit = "epoches"
	Unspecified Unit = "unspecified"
)

// ToProto converts the internal representation of a unit to protobuf.
func (u Unit) ToProto() experimentv1.TrainingLength_Unit {
	switch u {
	case Records:
		return experimentv1.TrainingLength_UNIT_RECORDS
	case Batches:
		return experimentv1.TrainingLength_UNIT_BATCHES
	case Epochs:
		return experimentv1.TrainingLength_UNIT_EPOCHS
	default:
		return experimentv1.TrainingLength_UNIT_UNSPECIFIED
	}
}

// UnitFromProto returns a model.Unit from its protobuf representation.
func UnitFromProto(u experimentv1.TrainingLength_Unit) Unit {
	switch u {
	case experimentv1.TrainingLength_UNIT_RECORDS:
		return Records
	case experimentv1.TrainingLength_UNIT_BATCHES:
		return Batches
	case experimentv1.TrainingLength_UNIT_EPOCHS:
		return Epochs
	default:
		return Unspecified
	}
}

// LengthV0 a training duration in terms of records, batches or epochs.
type LengthV0 struct {
	Unit  Unit
	Units int
}

// MarshalJSON implements the json.Marshaler interface.
func (l LengthV0) MarshalJSON() ([]byte, error) {
	switch l.Unit {
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
func (l *LengthV0) UnmarshalJSON(b []byte) error {
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

// ToProto converts a model.Length to its protobuf representation.
func (l LengthV0) ToProto() *experimentv1.TrainingLength {
	return &experimentv1.TrainingLength{
		Unit:   l.Unit.ToProto(),
		Length: int32(l.Units),
	}
}

// LengthFromProto returns a model.Length from its protobuf representation.
func LengthFromProto(l *experimentv1.TrainingLength) Length {
	return Length{
		Unit:  UnitFromProto(l.Unit),
		Units: int(l.Length),
	}
}

// NewLength returns a new length with the specified unit and length.
func NewLength(unit Unit, units int) LengthV0 {
	return LengthV0{Unit: unit, Units: units}
}

// NewLengthInRecords returns a new length in terms of records.
func NewLengthInRecords(records int) LengthV0 {
	return LengthV0{Unit: Records, Units: records}
}

// NewLengthInBatches returns a new length in terms of batches.
func NewLengthInBatches(batches int) LengthV0 {
	return LengthV0{Unit: Batches, Units: batches}
}

// NewLengthInEpochs returns a new LengthV0 in terms of epochs.
func NewLengthInEpochs(epochs int) LengthV0 {
	return LengthV0{Unit: Epochs, Units: epochs}
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
func (l LengthV0) MultInt(other int) LengthV0 {
	return NewLength(l.Unit, l.Units*other)
}

// Div divides a length by another length.
func (l LengthV0) Div(other LengthV0) LengthV0 {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units/other.Units)
}

// DivInt divides a length by an int.
func (l LengthV0) DivInt(other int) LengthV0 {
	return NewLength(l.Unit, l.Units/other)
}
