package model

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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

// PartialUnits represent partial epochs, batches or records where the Unit is implied.
type PartialUnits float64

// UnitContext contains all the context for switching the Unit of a Length freely.
type UnitContext struct {
	defaultUnit     expconf.Unit
	globalBatchSize int
	recordsPerEpoch int
}

// NewUnitContext creates a new UnitContext.
func NewUnitContext(defaultUnit expconf.Unit, globalBatchSize, recordsPerEpoch int) UnitContext {
	return UnitContext{defaultUnit, globalBatchSize, recordsPerEpoch}
}

// Length a training duration in terms of records, batches or epochs.
type Length struct {
	Unit  Unit
	Units int
}

// MarshalJSON implements the json.Marshaler interface.
func (l Length) MarshalJSON() ([]byte, error) {
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

// ToProto converts a model.Length to its protobuf representation.
func (l Length) ToProto() *experimentv1.TrainingLength {
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
func NewLength(unit Unit, units int) Length {
	return Length{Unit: unit, Units: units}
}

// NewLengthInRecords returns a new length in terms of records.
func NewLengthInRecords(records int) Length {
	return Length{Unit: Records, Units: records}
}

// NewLengthInBatches returns a new length in terms of batches.
func NewLengthInBatches(batches int) Length {
	return Length{Unit: Batches, Units: batches}
}

// NewLengthInEpochs returns a new Length in terms of epochs.
func NewLengthInEpochs(epochs int) Length {
	return Length{Unit: Epochs, Units: epochs}
}

// UnitsFromBatches return the number of units completed by the given batches, rounded up.
func UnitsFromBatches(ctx UnitContext, batches int) PartialUnits {
	switch ctx.defaultUnit {
	case expconf.Records:
		return PartialUnits(float64(batches * ctx.globalBatchSize))
	case expconf.Batches:
		return PartialUnits(float64(batches))
	case expconf.Epochs:
		return PartialUnits(float64(batches*ctx.globalBatchSize) / float64(ctx.recordsPerEpoch))
	default:
		panic(fmt.Sprintf("invalid unit in ctx: %s", ctx.defaultUnit))
	}
}

func (l Length) String() string {
	return fmt.Sprintf("%d %s", l.Units, l.Unit)
}

// Add adds a length to another length.
func (l Length) Add(other Length) Length {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units+other.Units)
}

// Sub subtracts a length from another length.
func (l Length) Sub(other Length) Length {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units-other.Units)
}

// Mult multiplies a length by another length.
func (l Length) Mult(other Length) Length {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units*other.Units)
}

// MultInt multiplies a length by an int.
func (l Length) MultInt(other int) Length {
	return NewLength(l.Unit, l.Units*other)
}

// Div divides a length by another length.
func (l Length) Div(other Length) Length {
	check.Panic(check.Equal(l.Unit, other.Unit))
	return NewLength(l.Unit, l.Units/other.Units)
}

// DivInt divides a length by an int.
func (l Length) DivInt(other int) Length {
	return NewLength(l.Unit, l.Units/other)
}

// ToNearestBatch converts a training length to the nearest batch, potentially truncating some units
// if they are provided as records or epochs.
func (u UnitContext) ToNearestBatch(l expconf.Length) int {
	switch l.Unit {
	case expconf.Records:
		return l.Units / u.globalBatchSize
	case expconf.Batches:
		return l.Units
	case expconf.Epochs:
		return (l.Units * u.recordsPerEpoch) / u.globalBatchSize
	default:
		panic(fmt.Sprintf("invalid Unit passed to unitsToBatches %s", l.Unit))
	}
}

// EqualWithinBatch returns true is the given length and batches are equal within one
// batch size.
func (u UnitContext) EqualWithinBatch(l expconf.Length, batches int) bool {
	switch l.Unit {
	case expconf.Records:
		diff := abs(l.Units - batches*u.globalBatchSize)
		return diff < u.globalBatchSize
	case expconf.Batches:
		return l.Units == batches
	case expconf.Epochs:
		diff := abs(l.Units*u.recordsPerEpoch - batches*u.globalBatchSize)
		return diff < u.globalBatchSize
	default:
		panic(fmt.Sprintf("invalid Unit passed to unitsToBatches %s", l.Unit))
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
