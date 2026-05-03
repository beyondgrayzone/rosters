package config

import "testing"

func TestConfigSchema(t *testing.T) {
	schema := ConfigSchema()
	if schema == nil {
		t.Fatal("ConfigSchema returned nil")
	}
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Error("missing or wrong $schema")
	}
	if schema["title"] != "Rosterss project config" {
		t.Error("missing title")
	}
	_, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not a map")
	}
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("required not a []string; got %T", schema["required"])
	}
	requiredFields := []string{"project", "version"}
	if len(required) != len(requiredFields) {
		t.Errorf("required length = %d, want %d", len(required), len(requiredFields))
	}
}
