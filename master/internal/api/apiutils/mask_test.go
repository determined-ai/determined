package apiutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	field_mask "google.golang.org/genproto/protobuf/field_mask"
)

func TestFieldInSet(t *testing.T) {
	t.Run("respects empty FieldMasks", func(t *testing.T) {
		m := NewFieldMask(nil)

		testPaths := []string{
			"",
			"a.b.c",
			"something.that.should.not.come.up",
			"averylongstringthatdoesnotcontainanydotswhichareusedtoseparatefields",
		}

		for _, p := range testPaths {
			require.True(t, m.FieldInSet(p), "all fields should be considered in the field set of"+
				" an empty FieldSet")
		}
	})

	t.Run("handles ancestors in FieldMask", func(t *testing.T) {
		protoMask := field_mask.FieldMask{
			Paths: []string{
				"a.b",
				"l.m.n.o.p",
				"x.y.z",
			},
		}
		mask := NewFieldMask(&protoMask)

		testCases := map[string]bool{
			"l.m.n":       false,
			"a.b":         true,
			"a.b.c":       true,
			"a.b.c.d.e.f": true,
			"x.y.z":       true,
			"x.y.z.aa":    true,
			"foo.bar":     false,
		}
		for path, expected := range testCases {
			found := mask.FieldInSet(path)
			require.Equal(t, expected, found, "got an unexpected result for path %s", path)
		}
	})
}
