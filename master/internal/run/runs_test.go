package run

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
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
	flattened, keyCount, err := FlattenRunMetadata(data)
	require.NoError(t, err)
	require.Equal(t, 8, keyCount)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:     0,
			FlatKey:   "key1",
			Value:     "1",
			DataType:  "integer",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key2",
			Value:     "2.1",
			DataType:  "float",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key3",
			Value:     "true",
			DataType:  "boolean",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key4",
			Value:     "string",
			DataType:  "string",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key5",
			Value:     "2021-01-01T00:00:00Z",
			DataType:  "timestamp",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key6",
			Value:     "2021-01-01T00:00:00Z",
			DataType:  "timestamp",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key7.key8",
			Value:     "3",
			DataType:  "integer",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key7",
			Value:     "string_1",
			DataType:  "string",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key7",
			Value:     "string_2",
			DataType:  "string",
			ProjectID: 0,
		},
	}, flattened)
}

func TestFlattenMetadataEmpty(t *testing.T) {
	data := map[string]interface{}{}
	_, _, err := FlattenRunMetadata(data)
	require.Error(t, err, "metadata is empty")
}

// TODO(corban): this should actually be adjusted to return an error since we don't want to
// allow empty post requests to be made to the metadata endpoint.
func TestFlattenMetadataNil(t *testing.T) {
	_, _, err := FlattenRunMetadata(nil)
	require.Error(t, err, "metadata is empty")
}

func TestFlattenMetadataNested(t *testing.T) {
	data := map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": 1,
		},
	}
	flattened, keyCount, err := FlattenRunMetadata(data)
	require.NoError(t, err)
	require.Equal(t, 2, keyCount)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:     0,
			FlatKey:   "key1.key2",
			Value:     "1",
			DataType:  "integer",
			ProjectID: 0,
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
	flattened, keyCount, err := FlattenRunMetadata(data)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:     0,
			FlatKey:   "key1",
			Value:     "1",
			DataType:  "integer",
			ProjectID: 0,
		},
		{
			RunID:     0,
			FlatKey:   "key1",
			Value:     "2",
			DataType:  "integer",
			ProjectID: 0,
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
	flattened, keyCount, err := FlattenRunMetadata(data)
	require.NoError(t, err)
	require.Equal(t, 2, keyCount)
	require.ElementsMatch(t, []model.RunMetadataIndex{
		{
			RunID:     0,
			FlatKey:   "key1.key2",
			Value:     "1",
			DataType:  "integer",
			ProjectID: 0,
		},
	}, flattened)
}

func TestMergeRunMetadataOverwriteFailure(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	data2 := map[string]interface{}{
		"key2": 3,
		"key3": 4,
	}
	_, err := db.MergeRunMetadata(data1, data2)
	require.Error(t, err)
}

func TestMergeRunMetadata(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	data2 := map[string]interface{}{
		"key3": 3,
		"key4": 4,
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
		"key4": 4,
	}, merged)
}

func TestMergeRunMetadataNested(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": 1,
		"key2": map[string]interface{}{
			"key3": 2,
		},
	}
	data2 := map[string]interface{}{
		"key2": map[string]interface{}{
			"key4": 3,
		},
		"key5": 4,
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": 1,
		"key2": map[string]interface{}{
			"key3": 2,
			"key4": 3,
		},
		"key5": 4,
	}, merged)
}

func TestMergeRunMetadataArrayAppend(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": []interface{}{
			1,
			2,
		},
	}
	data2 := map[string]interface{}{
		"key1": []interface{}{
			3,
		},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"key1": []interface{}{1, 2, 3}}, merged)
}

func TestMergeRunMetadataArray(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": []interface{}{
			1,
			2,
		},
	}
	data2 := map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": 3,
			"key3": 4,
			"key4": 5,
		},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": []interface{}{
			1,
			2,
			map[string]interface{}{
				"key2": 3,
				"key3": 4,
				"key4": 5,
			},
		},
	}, merged)
}

func TestMergeRunMetadataArrayNested(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": 1,
		},
	}
	data2 := map[string]interface{}{
		"key1": map[string]interface{}{
			"key3": 2,
		},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": map[string]interface{}{
			"key2": 1,
			"key3": 2,
		},
	}, merged)
}

func TestMergeRunMetadataArrayNestedList(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
		},
	}
	data2 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key3": 2,
			},
		},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
			map[string]interface{}{
				"key3": 2,
			},
		},
	}, merged)
}

func TestMergeRunMetadataArrayNestedListFailure(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
		},
	}
	data2 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 2,
			},
		},
	}
	_, err := db.MergeRunMetadata(data1, data2)
	require.ErrorContains(t, err, "attempts to overwrite existing entry ('key2': 1) with new value '2'")
}

func TestMergeRunMetadataArrayNestedListDifferentLength(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
		},
	}
	data2 := map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key3": 2,
			},
			map[string]interface{}{
				"key4": 3,
			},
		},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": []interface{}{
			map[string]interface{}{
				"key2": 1,
			},
			map[string]interface{}{
				"key3": 2,
			},
			map[string]interface{}{
				"key4": 3,
			},
		},
	}, merged)
}

func TestMergeRunMetadataAppendingToPrimitive(t *testing.T) {
	data1 := map[string]interface{}{
		"key1": 1,
	}
	data2 := map[string]interface{}{
		"key1": map[string]interface{}{"key2": 2},
	}
	merged, err := db.MergeRunMetadata(data1, data2)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"key1": []interface{}{
			1,
			map[string]interface{}{
				"key2": 2,
			},
		},
	}, merged)
}

func TestFlattenMetadataTooLongKey(t *testing.T) {
	data := map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	data[strings.Repeat("a", MaxMetadataKeyLength+1)] = 3
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
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
	_, _, err := FlattenRunMetadata(data)
	require.ErrorContains(
		t,
		err,
		fmt.Sprintf(
			"metadata exceeds maximum nesting depth of %d",
			MaxMetadataDepth,
		),
	)
}
