package check

import (
	"fmt"
	"reflect"
)

func isInterfaceNil(val interface{}) bool {
	return val == nil ||
		(reflect.ValueOf(val).Kind() == reflect.Ptr &&
			reflect.ValueOf(val).IsNil())
}

func internalFormat(original, indirect interface{}) string {
	if reflect.ValueOf(indirect).Kind() == reflect.Ptr && !isInterfaceNil(indirect) {
		return internalFormat(original, reflect.Indirect(reflect.ValueOf(indirect)).Interface())
	}
	if reflect.TypeOf(original) == reflect.TypeOf(indirect) {
		return fmt.Sprintf("%+v", original)
	}
	return fmt.Sprintf("%T(%+v)", original, indirect)
}

func format(i interface{}) string {
	return internalFormat(i, i)
}

func messageFromMsgAndArgs(formatPointers bool, msgAndArgs ...interface{}) string {
	switch {
	case len(msgAndArgs) == 1:
		switch msg := msgAndArgs[0].(type) {
		case string:
			return msg
		default:
			return fmt.Sprintf("%+v", format(msg))
		}
	case len(msgAndArgs) > 1:
		args := make([]interface{}, 0, len(msgAndArgs)-1)
		for _, arg := range msgAndArgs[1:] {
			if formatPointers {
				args = append(args, format(arg))
			} else {
				args = append(args, arg)
			}
		}
		return fmt.Sprintf(msgAndArgs[0].(string), args...)
	default:
		return ""
	}
}
