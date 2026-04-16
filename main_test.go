package testastic_test

import (
	"os"
	"testing"

	"github.com/monkescience/testastic"
)

var testCLI *testastic.Binary

func TestMain(m *testing.M) {
	testCLI = testastic.BuildBinaryMain(m, "./testdata/testcli")
	code := m.Run()

	testCLI.Cleanup()
	os.Exit(code)
}
