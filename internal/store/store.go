// Package store persists key→provider bindings to a JSON state file and serves
// lookups to the scheduler (by api-key hash) and the management API (CRUD).
package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DefaultStateFile is used when config does not specify state_file.
const DefaultStateFile = "key-bind-state.json"

// Binding ties one platform api-key to the set of providers/accounts it may use.
// Allow entries are either a provider name (e.g. "claude", "openrouter") or an
// explicit "auth:<id>" pinning a specific account.
type Binding struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyHash    string   `json:"key_hash"`         // sha256(apiKey), never the plaintext
	KeyPreview string   `json:"key_preview"`      // display-only mask, e.g. "sk-abcde...12345"
	Allow      []string `json:"allow"`
	Enabled    bool     `json:"enabled"`
}

type fileState struct {
	Bindings []Binding `json:"bindings"`
}

type Store struct {
	mu        sync.RWMutex
	statePath string
	byID      map[string]*Binding
	byHash    map[string]*Binding
}

func NewStore() *Store {
	return &Store{
		byID:   make(map[string]*Binding),
		byHash: make(map[string]*Binding),
	}
}

// HashKey returns the stable hash stored for an api-key. The scheduler hashes
// the key read from the request and looks the binding up by this value.
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// PreviewKey masks a key for display (keeps first 7 / last 5 chars).
func PreviewKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 12 {
		return key
	}
	return fmt.Sprintf("%s...%s", key[:7], key[len(key)-5:])
}

// ResolveStatePath resolves a possibly-relative state file path to absolute.
func ResolveStatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultStateFile
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// Configure (re)sets the state file path and reloads bindings from disk. Called
// on plugin.register / plugin.reconfigure.
func (s *Store) Configure(statePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statePath = statePath
	return s.reloadLocked()
}

func (s *Store) reloadLocked() error {
	s.byID = make(map[string]*Binding)
	s.byHash = make(map[string]*Binding)
	if s.statePath == "" {
		return nil
	}
	raw, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // first run: no state yet
		}
		return err
	}
	var st fileState
	if err := json.Unmarshal(raw, &st); err != nil {
		return err
	}
	for i := range st.Bindings {
		b := st.Bindings[i]
		s.indexLocked(&b)
	}
	return nil
}

func (s *Store) indexLocked(b *Binding) {
	s.byID[b.ID] = b
	if b.KeyHash != "" {
		if _, exists := s.byHash[b.KeyHash]; !exists {
			s.byHash[b.KeyHash] = b
		}
	}
}

// saveLocked serializes current bindings and atomically writes the state file.
func (s *Store) saveLocked() error {
	if s.statePath == "" {
		return fmt.Errorf("state file not configured")
	}
	st := fileState{Bindings: make([]Binding, 0, len(s.byID))}
	// Deterministic order by ID for stable diffs.
	for _, b := range s.byID {
		st.Bindings = append(st.Bindings, *b)
	}
	sortBindings(st.Bindings)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(s.statePath, data)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	return os.Rename(tempName, path)
}

// --- queries ---

// FindByKeyHash returns the enabled-or-not binding for a key hash, or nil.
func (s *Store) FindByKeyHash(hash string) *Binding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byHash[hash]
}

// List returns a snapshot of all bindings ordered by ID.
func (s *Store) List() []Binding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Binding, 0, len(s.byID))
	for _, b := range s.byID {
		out = append(out, *b)
	}
	sortBindings(out)
	return out
}

// --- mutations (each persists) ---

// Create inserts a new binding, generating its ID and hashing the plaintext key.
func (s *Store) Create(name, plaintextKey string, allow []string, enabled bool) (Binding, error) {
	plaintextKey = strings.TrimSpace(plaintextKey)
	if plaintextKey == "" {
		return Binding{}, fmt.Errorf("key is required")
	}
	id, err := newID()
	if err != nil {
		return Binding{}, err
	}
	b := Binding{
		ID:         id,
		Name:       strings.TrimSpace(name),
		KeyHash:    HashKey(plaintextKey),
		KeyPreview: PreviewKey(plaintextKey),
		Allow:      normalizeAllow(allow),
		Enabled:    enabled,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byHash[b.KeyHash]; exists {
		return Binding{}, fmt.Errorf("a binding for this key already exists")
	}
	s.indexLocked(&b)
	if err := s.saveLocked(); err != nil {
		return Binding{}, err
	}
	return b, nil
}

// Update modifies an existing binding. PlaintextKey empty => keep existing key.
func (s *Store) Update(id, name, plaintextKey string, allow []string, enabled *bool) (Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[id]
	if !ok {
		return Binding{}, fmt.Errorf("binding not found: %s", id)
	}
	updated := *cur
	if name != "" {
		updated.Name = strings.TrimSpace(name)
	}
	if plaintextKey != "" {
		plaintextKey = strings.TrimSpace(plaintextKey)
		updated.KeyHash = HashKey(plaintextKey)
		updated.KeyPreview = PreviewKey(plaintextKey)
	}
	if allow != nil {
		updated.Allow = normalizeAllow(allow)
	}
	if enabled != nil {
		updated.Enabled = *enabled
	}
	// Rebuild byHash if the key changed: a key maps to at most one binding.
	if updated.KeyHash != cur.KeyHash {
		for h, b := range s.byHash {
			if b == cur {
				delete(s.byHash, h)
			}
		}
	}
	*cur = updated
	s.indexLocked(cur)
	if err := s.saveLocked(); err != nil {
		return Binding{}, err
	}
	return *cur, nil
}

// Delete removes a binding by id.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("binding not found: %s", id)
	}
	delete(s.byID, id)
	for h, bb := range s.byHash {
		if bb == b {
			delete(s.byHash, h)
		}
	}
	return s.saveLocked()
}

func normalizeAllow(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

func newID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "kb_" + hex.EncodeToString(buf), nil
}

// sortBindings orders bindings by ID for stable output/diffs.
func sortBindings(bs []Binding) {
	sort.Slice(bs, func(i, j int) bool { return bs[i].ID < bs[j].ID })
}
