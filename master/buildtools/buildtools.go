package main

import (
	// These are phony imports of command packages so that go module commands know that we care about
	// these packages.
	_ "github.com/rakyll/gotest"
	_ "golang.org/x/tools/cmd/goimports"
)
