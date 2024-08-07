package run

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestFlattenMetadata(t *testing.T) {
	data := map[string]interface{}{
		"key1": 1,
		"key2": 2.1,
		"key3": true,
		"key4": "string",
		"key5": "2021-01-01",
		"key6": "2021-01",
		"key7": []interface{}{
			map[string]interface{}{
				"key8": 3,
			},
			[]interface{}{
				"string_1",
				"string_2",
			},
		},
	}
	flattened, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:          0,
			FlatKey:        "key1",
			IntegerValue:   ptrs.Ptr(1),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key2",
			FloatValue:     ptrs.Ptr(2.1),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key3",
			BooleanValue:   ptrs.Ptr(true),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key4",
			StringValue:    ptrs.Ptr("string"),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key5",
			TimestampValue: ptrs.Ptr("2021-01-01T00:00:00Z"),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key6",
			TimestampValue: ptrs.Ptr("2021-01-01T00:00:00Z"),
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key7.key8",
			IntegerValue:   ptrs.Ptr(3),
			ProjectID:      0,
			IsArrayElement: true,
		},
		{
			RunID:          0,
			FlatKey:        "key7",
			StringValue:    ptrs.Ptr("string_1"),
			ProjectID:      0,
			IsArrayElement: true,
		},
		{
			RunID:          0,
			FlatKey:        "key7",
			StringValue:    ptrs.Ptr("string_2"),
			ProjectID:      0,
			IsArrayElement: true,
		},
	}, flattened)
}

func TestFlattenMetadataEmpty(t *testing.T) {
	data := map[string]interface{}{}
	flatMetadata, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.Empty(t, flatMetadata)
}

func TestFlattenMetadataNil(t *testing.T) {
	flatMetadata, err := FlattenMetadata(nil)
	require.NoError(t, err)
	require.Empty(t, flatMetadata)
}

func TestFlattenMetadataNested(t *testing.T) {
	data := map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": 1,
		},
	}
	flattened, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:          0,
			FlatKey:        "key1.key2",
			IntegerValue:   ptrs.Ptr(1),
			ProjectID:      0,
			IsArrayElement: false,
		},
	}, flattened)
}

func TestFlattenMetadataArray(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			1,
			2,
		},
	}
	flattened, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:          0,
			FlatKey:        "key1",
			IntegerValue:   ptrs.Ptr(1),
			ProjectID:      0,
			IsArrayElement: true,
		},
		{
			RunID:          0,
			FlatKey:        "key1",
			IntegerValue:   ptrs.Ptr(2),
			ProjectID:      0,
			IsArrayElement: true,
		},
	}, flattened)
}

func TestFlattenMetadataArrayNested(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
		},
	}
	flattened, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:          0,
			FlatKey:        "key1.key2",
			IntegerValue:   ptrs.Ptr(1),
			ProjectID:      0,
			IsArrayElement: true,
		},
	}, flattened)
}

func TestFlattenMetadataTooLongKey(t *testing.T) {
	data := map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	data[strings.Repeat("a", MaxMetadataKeyLength+1)] = 3
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata key exceeds maximum length of %d characters",
			MaxMetadataKeyLength,
		),
	)
}

func TestFlattenMetadataTooLongArray(t *testing.T) {
	data := map[string]interface{}{}
	tooLongArray := make([]interface{}, MaxMetadataArrayLength+1)
	for i := range tooLongArray {
		tooLongArray[i] = i
	}
	data["key1"] = tooLongArray
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata array exceeds maximum length of %d/%d elements",
			len(tooLongArray), MaxMetadataArrayLength,
		),
	)
}

func TestFlattenMetadataTooLongNestedArray(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			1,
			2,
			3,
			4,
			5,
		},
	}
	nestedArray := make([]interface{}, MaxMetadataArrayLength+1)
	for i := range nestedArray {
		nestedArray[i] = i
	}
	data["key1"].([]interface{})[0] = nestedArray
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata array exceeds maximum length of %d/%d elements",
			len(nestedArray),
			MaxMetadataArrayLength,
		),
	)
}

func TestFlattenMetadataTooLongValue(t *testing.T) {
	data := map[string]interface{}{
		"key1": strings.Repeat("a", MaxMetadataValueStringLength+1),
	}
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata value exceeds maximum length of %d characters",
			MaxMetadataValueStringLength,
		),
	)
}

func TestFlattenMetadataArrayElementTooLongValue(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			strings.Repeat("a", MaxMetadataValueStringLength+1),
		},
	}
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf("metadata value exceeds maximum length of %d characters",
			MaxMetadataValueStringLength,
		),
	)
}

func TestFlattenMetadataArrayElementTooLongKey(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			1,
		},
	}
	data[strings.Repeat("a", MaxMetadataKeyLength+1)] = 2
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata key exceeds maximum length of %d characters",
			MaxMetadataKeyLength,
		),
	)
}

func TestFlattenMetadataArrayElementTooLongKeyInNestedArray(t *testing.T) {
	data := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
		},
	}
	data["key1"].([]interface{})[0].(map[string]interface{})[strings.Repeat("a", MaxMetadataKeyLength+1)] = 2
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf("metadata key exceeds maximum length of %d characters",
			MaxMetadataKeyLength,
		),
	)
}

func TestFlattenMetadataNestingTooDeep(t *testing.T) {
	data := map[string]interface{}{}
	current := data
	for i := 0; i < MaxMetadataDepth+1; i++ {
		key := fmt.Sprintf("key_%d", i)
		current[key] = map[string]interface{}{"key_level": i}
		current = current[key].(map[string]interface{})
	}
	_, err := FlattenMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata exceeds maximum nesting depth of %d",
			MaxMetadataDepth,
		),
	)
}

func TestFlattenMetadataEmptyNestedStructs(t *testing.T) {
	data := map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": map[string]interface{}{},
			"key3": []any{},
		},
	}
	flattened, err := FlattenMetadata(data)
	require.NoError(t, err)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:          0,
			FlatKey:        "key1.key2",
			ProjectID:      0,
			IsArrayElement: false,
		},
		{
			RunID:          0,
			FlatKey:        "key1.key3",
			ProjectID:      0,
			IsArrayElement: true,
		},
	}, flattened)
}
