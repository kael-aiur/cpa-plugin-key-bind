package plugin

import (
	"encoding/json"
	"net/http"
	"testing"

	"cpa-plugin-key-bind/internal/bindings"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	return NewApp()
}

func configureTestBinding(t *testing.T, app *App, id, name, key string, allow []string, enabled bool) {
	t.Helper()
	enabledValue := enabled
	next, err := bindings.Build([]bindings.ConfigBinding{
		{
			ID:         id,
			Name:       name,
			KeyHash:    bindings.HashKey(key),
			KeyPreview: bindings.PreviewKey(key),
			Allow:      allow,
			Enabled:    &enabledValue,
		},
	})
	if err != nil {
		t.Fatalf("build test bindings: %v", err)
	}
	app.mu.Lock()
	app.index = next
	app.mu.Unlock()
}

func callPick(t *testing.T, a *App, req SchedulerPickRequest) Envelope {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	response, err := a.pickAuth(raw)
	if err != nil {
		t.Fatalf("pickAuth: %v", err)
	}
	var env Envelope
	if err := json.Unmarshal(response, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return env
}

func TestPickAuthTrustsHostFilteredCandidateRegardlessOfGlobalStatus(t *testing.T) {
	a := newTestApp(t)
	const key = "sk-test-status"
	const authID = "codex-user@example.com-team.json"
	configureTestBinding(t, a, "kb_3123456789abcdef01234567", "team", key, []string{"auth:" + authID}, true)

	env := callPick(t, a, SchedulerPickRequest{
		Model: "gpt-5.6-terra",
		Options: SchedulerPickOptions{
			Headers: map[string][]string{"Authorization": {"Bearer " + key}},
		},
		Candidates: []SchedulerAuthCandidate{
			{
				ID:       authID,
				Provider: "codex",
				// The host already decided this auth is available for the current
				// model. A global error caused by another model must not make the
				// binding plugin reject it.
				Status: "error",
			},
		},
	})
	if !env.OK {
		t.Fatalf("expected success envelope, got error: %+v", env.Error)
	}
	var pick SchedulerPickResponse
	if err := json.Unmarshal(env.Result, &pick); err != nil {
		t.Fatalf("unmarshal pick response: %v", err)
	}
	if !pick.Handled || pick.AuthID != authID {
		t.Fatalf("unexpected pick: %+v", pick)
	}
}

func TestPickAuthWithoutBindingDefersToHost(t *testing.T) {
	a := newTestApp(t)
	env := callPick(t, a, SchedulerPickRequest{
		Options: SchedulerPickOptions{
			Headers: map[string][]string{"Authorization": {"Bearer sk-unbound"}},
		},
		Candidates: []SchedulerAuthCandidate{{ID: "any", Provider: "codex", Status: "active"}},
	})
	if !env.OK {
		t.Fatalf("expected success envelope, got error: %+v", env.Error)
	}
	var pick SchedulerPickResponse
	if err := json.Unmarshal(env.Result, &pick); err != nil {
		t.Fatalf("unmarshal pick response: %v", err)
	}
	if pick.Handled {
		t.Fatalf("unbound key must defer to host: %+v", pick)
	}
}

func TestPickAuthRejectsWhenNoAllowedCandidateRemains(t *testing.T) {
	a := newTestApp(t)
	const key = "sk-test-no-match"
	configureTestBinding(t, a, "kb_4123456789abcdef01234567", "team", key, []string{"auth:allowed.json"}, true)

	env := callPick(t, a, SchedulerPickRequest{
		Options: SchedulerPickOptions{
			Headers: map[string][]string{"Authorization": {"Bearer " + key}},
		},
		Candidates: []SchedulerAuthCandidate{{ID: "other.json", Provider: "codex", Status: "active"}},
	})
	if env.OK || env.Error == nil {
		t.Fatalf("expected auth_not_found envelope, got: %+v", env)
	}
	if env.Error.Code != "auth_not_found" || env.Error.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("unexpected error: %+v", env.Error)
	}
}
