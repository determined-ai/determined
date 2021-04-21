package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

// Unit is the type of unit for specifying lengths.
type Unit string

// All the units available for lengths.
const (
	Records Unit = "records"
	Batches Unit = "batches"
	Epochs  Unit = "epoches"
)

// UnitContext contains all the context for switching the Unit of a Length freely.
type UnitContext struct {
	defaultUnit     Unit
	globalBatchSize int
	recordsPerEpoch int
}

// NewUnitContext creates a new UnitContext.
func NewUnitContext(defaultUnit Unit, globalBatchSize, recordsPerEpoch int) UnitContext {
	return UnitContext{defaultUnit, globalBatchSize, recordsPerEpoch}
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

// UnitsFromBatches return the number of units completed by the given batches, rounded up.
func UnitsFromBatches(batches int, ctx UnitContext) float64 {
	switch ctx.defaultUnit {
	case Records:
		return float64(batches * ctx.globalBatchSize)
	case Batches:
		return float64(batches)
	case Epochs:
		return float64(batches*ctx.globalBatchSize) / float64(ctx.recordsPerEpoch)
	default:
		panic(fmt.Sprintf("invalid unit in ctx: %s", ctx.defaultUnit))
	}
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

// ToNearestBatch converts a training length to the nearest batch, potentially truncating some units
// if they are provided as records or epochs.
func (l LengthV0) ToNearestBatch(ctx UnitContext) int {
	switch l.Unit {
	case Records:
		return l.Units / ctx.globalBatchSize
	case Batches:
		return l.Units
	case Epochs:
		return (l.Units * ctx.recordsPerEpoch) / ctx.globalBatchSize
	default:
		panic(fmt.Sprintf("invalid Unit passed to unitsToBatches %s", l.Unit))
	}
}

// EqualWithinBatch returns true is the given length and batches are equal within one
// batch size.
func (l LengthV0) EqualWithinBatch(batches int, ctx UnitContext) bool {
	switch l.Unit {
	case Records:
		diff := abs(l.Units - batches*ctx.globalBatchSize)
		return diff < ctx.globalBatchSize
	case Batches:
		return l.Units == batches
	case Epochs:
		diff := abs(l.Units*ctx.recordsPerEpoch - batches*ctx.globalBatchSize)
		return diff < ctx.globalBatchSize
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
