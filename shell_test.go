package tish

import (
	"testing"
)

func TestShell(t *testing.T) {
	testAssign("assignment", testAssign)
}

func testAssign(t *testing.T) {
	t.SkipNow()
}
