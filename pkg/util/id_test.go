package util

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	prefix := "test"
	existing := []string{"test-1234"}

	t.Run("generates new ID", func(t *testing.T) {
		id := GenerateID(prefix, existing)
		if !strings.HasPrefix(id, "test-") {
			t.Errorf("expected prefix test-, got %s", id)
		}
		if id == "test-1234" {
			t.Error("should not generate existing ID")
		}
	})

	t.Run("handles collisions", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := GenerateID(prefix, nil)
			if ids[id] {
				t.Errorf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})
}
