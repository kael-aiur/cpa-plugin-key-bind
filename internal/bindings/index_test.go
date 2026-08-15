package bindings

import (
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func validConfig(id, hash string) ConfigBinding {
	return ConfigBinding{
		ID:         id,
		Name:       " Team A ",
		KeyHash:    hash,
		KeyPreview: "sk-test...value",
		Allow:      []string{" claude ", "", "claude", "auth:a.json"},
		Enabled:    boolPtr(true),
	}
}

func TestBuildNormalizesAndIndexesBindings(t *testing.T) {
	cfg := validConfig(
		"kb_0123456789abcdef01234567",
		"sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0",
	)

	index, err := Build([]ConfigBinding{cfg})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, ok := index.FindByKeyHash(cfg.KeyHash)
	if !ok {
		t.Fatal("expected hash lookup to succeed")
	}
	if got.Name != "Team A" {
		t.Fatalf("unexpected name: %q", got.Name)
	}
	if strings.Join(got.Allow, ",") != "claude,auth:a.json" {
		t.Fatalf("unexpected allow list: %#v", got.Allow)
	}
}

func TestBuildDefaultsMissingEnabledToTrue(t *testing.T) {
	cfg := validConfig(
		"kb_1123456789abcdef01234567",
		"sha256:13ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
	)
	cfg.Enabled = nil

	index, err := Build([]ConfigBinding{cfg})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := index.List()
	if len(got) != 1 || !got[0].Enabled {
		t.Fatalf("missing enabled must default true: %#v", got)
	}
}

func TestBuildRejectsDuplicateID(t *testing.T) {
	first := validConfig(
		"kb_2123456789abcdef01234567",
		"sha256:23ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
	)
	second := validConfig(first.ID,
		"sha256:33ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
	)

	_, err := Build([]ConfigBinding{first, second})
	if err == nil || !strings.Contains(err.Error(), "bindings[1].id") {
		t.Fatalf("expected indexed duplicate-id error, got %v", err)
	}
}

func TestBuildRejectsDuplicateHash(t *testing.T) {
	first := validConfig(
		"kb_3123456789abcdef01234567",
		"sha256:43ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
	)
	second := validConfig("kb_4123456789abcdef01234567", first.KeyHash)

	_, err := Build([]ConfigBinding{first, second})
	if err == nil || !strings.Contains(err.Error(), "bindings[1].key_hash") {
		t.Fatalf("expected indexed duplicate-hash error, got %v", err)
	}
}

func TestBuildRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ConfigBinding)
		want string
	}{
		{"id", func(b *ConfigBinding) { b.ID = "bad" }, "bindings[0].id"},
		{"hash", func(b *ConfigBinding) { b.KeyHash = "plaintext" }, "bindings[0].key_hash"},
		{"preview", func(b *ConfigBinding) { b.KeyPreview = "" }, "bindings[0].key_preview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(
				"kb_5123456789abcdef01234567",
				"sha256:53ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
			)
			tt.edit(&cfg)
			_, err := Build([]ConfigBinding{cfg})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestHashAndPreviewMatchLegacyBehavior(t *testing.T) {
	if got, want := HashKey(" sk-test "), "sha256:f3abf2a6cc4f00987743db5f544ba345b4899ae31f326d8ee9c4816de153c9e0"; got != want {
		t.Fatalf("HashKey = %q, want %q", got, want)
	}
	if got, want := PreviewKey("1234567890123"), "1234567...90123"; got != want {
		t.Fatalf("PreviewKey = %q, want %q", got, want)
	}
}

func TestEmptyIndexAndListSnapshot(t *testing.T) {
	empty := Empty()
	if got := empty.List(); len(got) != 0 {
		t.Fatalf("empty index returned %#v", got)
	}

	cfg := validConfig(
		"kb_6123456789abcdef01234567",
		"sha256:63ecb3f6f84b48f9a0515a3f79b28e61f8b2202de3a173c0f0daba50a1953649",
	)
	index, err := Build([]ConfigBinding{cfg})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	first := index.List()
	first[0].Name = "mutated"
	if second := index.List(); second[0].Name == "mutated" {
		t.Fatal("List must return a snapshot")
	}
}
