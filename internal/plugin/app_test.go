package plugin

import (
	"encoding/json"
	"testing"

	"cpa-plugin-key-bind/internal/bindings"
)

func lifecycleRequest(t *testing.T, yamlText string) []byte {
	t.Helper()
	raw, err := json.Marshal(LifecycleRequest{ConfigYAML: []byte(yamlText)})
	if err != nil {
		t.Fatalf("marshal lifecycle request: %v", err)
	}
	return raw
}

func TestConfigureLoadsBindingsFromHostConfig(t *testing.T) {
	app := NewApp()
	key := "sk-test"
	err := app.configure(lifecycleRequest(t, `
bindings:
  - id: kb_0123456789abcdef01234567
    name: Team A
    key_hash: `+bindings.HashKey(key)+`
    key_preview: sk-test
    allow: [codex]
    enabled: true
`))
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	got, ok := app.activeBindings().FindByKeyHash(bindings.HashKey(key))
	if !ok || got.Name != "Team A" {
		t.Fatalf("binding not active: %#v, %v", got, ok)
	}
}

func TestInvalidReconfigureKeepsPreviousBindings(t *testing.T) {
	app := NewApp()
	key := "sk-stable"
	valid := lifecycleRequest(t, `
bindings:
  - id: kb_1123456789abcdef01234567
    name: Stable
    key_hash: `+bindings.HashKey(key)+`
    key_preview: sk-stable
    allow: [claude]
    enabled: true
`)
	if err := app.configure(valid); err != nil {
		t.Fatalf("initial configure: %v", err)
	}

	invalid := lifecycleRequest(t, `
bindings:
  - id: invalid
    key_hash: plaintext
    key_preview: bad
    allow: []
    enabled: true
`)
	if err := app.configure(invalid); err == nil {
		t.Fatal("expected invalid reconfigure to fail")
	}
	if _, ok := app.activeBindings().FindByKeyHash(bindings.HashKey(key)); !ok {
		t.Fatal("invalid reconfigure replaced the previous valid index")
	}
}

func TestEmptyBindingsClearsActiveIndex(t *testing.T) {
	app := NewApp()
	key := "sk-clear"
	if err := app.configure(lifecycleRequest(t, `
bindings:
  - id: kb_2123456789abcdef01234567
    key_hash: `+bindings.HashKey(key)+`
    key_preview: sk-clear
    allow: [claude]
    enabled: true
`)); err != nil {
		t.Fatalf("initial configure: %v", err)
	}
	if err := app.configure(lifecycleRequest(t, "bindings: []\n")); err != nil {
		t.Fatalf("clear configure: %v", err)
	}
	if got := app.activeBindings().List(); len(got) != 0 {
		t.Fatalf("expected empty index, got %#v", got)
	}
}

func TestRegistrationDeclaresBindingsArrayField(t *testing.T) {
	registration := NewApp().registration()
	if len(registration.Metadata.ConfigFields) != 1 {
		t.Fatalf("unexpected fields: %#v", registration.Metadata.ConfigFields)
	}
	field := registration.Metadata.ConfigFields[0]
	if field.Name != "bindings" || field.Type != "array" {
		t.Fatalf("unexpected field: %#v", field)
	}
}
