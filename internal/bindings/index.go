package bindings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	bindingIDPattern = regexp.MustCompile(`^kb_[a-f0-9]{24}$`)
	keyHashPattern   = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

type ConfigBinding struct {
	ID         string   `yaml:"id" json:"id"`
	Name       string   `yaml:"name" json:"name"`
	KeyHash    string   `yaml:"key_hash" json:"key_hash"`
	KeyPreview string   `yaml:"key_preview" json:"key_preview"`
	Allow      []string `yaml:"allow" json:"allow"`
	Enabled    *bool    `yaml:"enabled" json:"enabled"`
}

type Binding struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyHash    string   `json:"key_hash"`
	KeyPreview string   `json:"key_preview"`
	Allow      []string `json:"allow"`
	Enabled    bool     `json:"enabled"`
}

type Index struct {
	byID   map[string]Binding
	byHash map[string]Binding
}

func Empty() *Index {
	return &Index{byID: map[string]Binding{}, byHash: map[string]Binding{}}
}

func Build(configs []ConfigBinding) (*Index, error) {
	index := Empty()
	for position, cfg := range configs {
		binding, err := normalizeBinding(position, cfg)
		if err != nil {
			return nil, err
		}
		if _, exists := index.byID[binding.ID]; exists {
			return nil, fmt.Errorf("bindings[%d].id: duplicate binding id", position)
		}
		if _, exists := index.byHash[binding.KeyHash]; exists {
			return nil, fmt.Errorf("bindings[%d].key_hash: duplicate key hash", position)
		}
		index.byID[binding.ID] = binding
		index.byHash[binding.KeyHash] = binding
	}
	return index, nil
}

func normalizeBinding(position int, cfg ConfigBinding) (Binding, error) {
	id := strings.TrimSpace(cfg.ID)
	if !bindingIDPattern.MatchString(id) {
		return Binding{}, fmt.Errorf("bindings[%d].id: invalid binding id", position)
	}
	keyHash := strings.TrimSpace(cfg.KeyHash)
	if !keyHashPattern.MatchString(keyHash) {
		return Binding{}, fmt.Errorf("bindings[%d].key_hash: expected sha256:<64 lowercase hex chars>", position)
	}
	preview := strings.TrimSpace(cfg.KeyPreview)
	if preview == "" || len(preview) > 128 {
		return Binding{}, fmt.Errorf("bindings[%d].key_preview: must contain 1-128 characters", position)
	}
	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	return Binding{
		ID:         id,
		Name:       strings.TrimSpace(cfg.Name),
		KeyHash:    keyHash,
		KeyPreview: preview,
		Allow:      normalizeAllow(cfg.Allow),
		Enabled:    enabled,
	}, nil
}

func (i *Index) FindByKeyHash(hash string) (Binding, bool) {
	if i == nil {
		return Binding{}, false
	}
	binding, ok := i.byHash[hash]
	if !ok {
		return Binding{}, false
	}
	binding.Allow = append([]string(nil), binding.Allow...)
	return binding, true
}

func (i *Index) List() []Binding {
	if i == nil {
		return []Binding{}
	}
	out := make([]Binding, 0, len(i.byID))
	for _, binding := range i.byID {
		binding.Allow = append([]string(nil), binding.Allow...)
		out = append(out, binding)
	}
	sort.Slice(out, func(left, right int) bool { return out[left].ID < out[right].ID })
	return out
}

func HashKey(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func PreviewKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 12 {
		return key
	}
	return fmt.Sprintf("%s...%s", key[:7], key[len(key)-5:])
}

func normalizeAllow(input []string) []string {
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, entry := range input {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, exists := seen[entry]; exists {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return out
}
