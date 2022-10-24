//go:build integration
// +build integration

package db

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}
