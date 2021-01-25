package schemas

import (
	"reflect"
)

// UnionDefaultSchema is a helper function for defining DefaultSchema on union-like objects.
// It searches for the non-nil union memeber and uses that member to define defaults for the common
// fields.  In short it turns this:
//
//     func (c *CheckpointStorageConfigV0) DefaultSource {
//         if c != nil {
//             if c.SharedFSConfig != nil {
//                 return c.SharedFSConfig.DefaultSource
//             }
//             if c.S3Config != nil {
//                 return c.S3Config.DefaultSource
//             }
//             if c.GCSConfig != nil {
//                 return c.GCSConfig.DefaultSource
//             }
//             if c.HDFSConfig != nil {
//                 return c.HDFSConfig.DefaultSource
//             }
//         }
//         return nil
//     }
//
// Into this:
//
//     func (c *CheckpointStorageConfigV0) DefaultSource() interface{} {
//         return schemas.UnionDefaultSchema(c)
//     }
func UnionDefaultSchema(in interface{}) interface{} {
	v := reflect.ValueOf(in)
	var ok bool
	if v, ok = derefInput(v); !ok {
		return nil
	}
	// Iterate through all the fields of the struct.
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if _, ok := fieldType.Tag.Lookup("union"); !ok {
			// This field has no "union" tag and cannot provide defaults.
			continue
		}

		field := v.Field(i)

		if _, ok := derefInput(field); !ok {
			// nil pointers cannot provide defaults.
			continue
		}

		// Get a source of defaults from a Defaultable or Schema interface.
		if defaultable, ok := field.Interface().(Defaultable); ok {
			return defaultable.DefaultSource()
		}else if schema, ok := field.Interface().(Schema); ok {
			return schema.ParsedSchema()
		}
	}
	return nil
}
