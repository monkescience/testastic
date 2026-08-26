//nolint:testpackage // The private YAML stream context is its intended test surface.
package testastic

import (
	"reflect"
	"testing"
)

func TestYAMLStreamContext_DocumentIdentity(t *testing.T) {
	t.Parallel()

	base := &config{}
	tests := []struct {
		name          string
		expectedCount int
		actualCount   int
		index         int
		path          string
		sameConfig    bool
	}{
		{name: "empty stream", path: "$", sameConfig: true},
		{name: "one actual document", actualCount: 1, path: "$", sameConfig: true},
		{name: "one expected document", expectedCount: 1, path: "$", sameConfig: true},
		{name: "matching single documents", expectedCount: 1, actualCount: 1, path: "$", sameConfig: true},
		{name: "two actual documents", actualCount: 2, path: "$[0]"},
		{name: "two expected documents", expectedCount: 2, path: "$[0]"},
		{name: "extra actual document", expectedCount: 1, actualCount: 2, index: 1, path: "$[1]"},
		{name: "missing actual document", expectedCount: 2, actualCount: 1, index: 1, path: "$[1]"},
		{name: "matching document streams", expectedCount: 2, actualCount: 2, index: 1, path: "$[1]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			stream := newYAMLStreamContext(test.expectedCount, test.actualCount, base)
			document := stream.document(test.index)

			if document.path != test.path {
				t.Errorf("path = %q, want %q", document.path, test.path)
			}

			if (document.config == base) != test.sameConfig {
				t.Errorf("config identity = %t, want %t", document.config == base, test.sameConfig)
			}
		})
	}
}

func TestYAMLStreamContext_DocumentConfig(t *testing.T) {
	t.Parallel()

	base := &config{
		IgnoreArrayOrder:      true,
		IgnoreArrayOrderPaths: []string{"$", "$.items", "$[0].roles", "id", "$invalid"},
		IgnoredFields:         []string{"$", "$.items", "$[0].roles", "id", "$invalid"},
		Update:                true,
		Message:               "context",
	}
	wantBasePaths := []string{"$", "$.items", "$[0].roles", "id", "$invalid"}
	wantSecondPaths := []string{"$[1]", "$[1].items", "$[1][0].roles", "id", "$invalid"}
	stream := newYAMLStreamContext(2, 2, base)
	first := stream.document(0)
	second := stream.document(1)

	if !reflect.DeepEqual(second.config.IgnoreArrayOrderPaths, wantSecondPaths) {
		t.Errorf("IgnoreArrayOrderPaths = %q, want %q", second.config.IgnoreArrayOrderPaths, wantSecondPaths)
	}

	if !reflect.DeepEqual(second.config.IgnoredFields, wantSecondPaths) {
		t.Errorf("IgnoredFields = %q, want %q", second.config.IgnoredFields, wantSecondPaths)
	}

	if !second.config.IgnoreArrayOrder || !second.config.Update || second.config.Message != "context" {
		t.Errorf("nonpath configuration was not preserved: %#v", second.config)
	}

	first.config.IgnoreArrayOrderPaths[0] = "changed"
	first.config.IgnoredFields[0] = "changed"

	if !reflect.DeepEqual(base.IgnoreArrayOrderPaths, wantBasePaths) ||
		!reflect.DeepEqual(base.IgnoredFields, wantBasePaths) {
		t.Errorf("base config was mutated: %#v", base)
	}

	if !reflect.DeepEqual(second.config.IgnoreArrayOrderPaths, wantSecondPaths) ||
		!reflect.DeepEqual(second.config.IgnoredFields, wantSecondPaths) {
		t.Errorf("document configs share qualified slices: %#v", second.config)
	}
}
