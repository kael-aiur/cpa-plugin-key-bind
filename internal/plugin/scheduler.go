package plugin

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"

	"cpa-plugin-key-bind/internal/store"
)

// roundRobin rotates across the filtered candidate set. SchedulerPickResponse
// returns a single AuthID, so load balancing within the allowed set is in-plugin.
var roundRobin uint64

// pickAuth implements scheduler.pick: narrow the candidate auth set to what the
// caller's api-key is allowed to use, then pick one.
func (a *App) pickAuth(raw []byte) ([]byte, error) {
	var req SchedulerPickRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}

	apiKey := extractAPIKey(req.Options.Headers)
	if apiKey == "" {
		// No recognizable caller key (e.g. internal host call). Do not interfere.
		return OKEnvelope(SchedulerPickResponse{Handled: false})
	}

	b := a.store.FindByKeyHash(store.HashKey(apiKey))
	if b == nil || !b.Enabled {
		// No active binding for this key -> defer to the host scheduler, i.e. the
		// platform's original strategy. (Requirement: "skip when no binding".)
		return OKEnvelope(SchedulerPickResponse{Handled: false})
	}
	if len(b.Allow) == 0 {
		return ErrorEnvelope("no_allowed_provider", "key-bind: this key has an empty allow list", http.StatusServiceUnavailable), nil
	}

	filtered := make([]SchedulerAuthCandidate, 0, len(req.Candidates))
	for _, cand := range req.Candidates {
		if !candidateUsable(cand.Status) {
			continue
		}
		if candidateAllowed(b.Allow, cand) {
			filtered = append(filtered, cand)
		}
	}
	if len(filtered) == 0 {
		// Honor isolation: never silently fall back to other providers/accounts.
		return ErrorEnvelope("auth_not_found", "key-bind: no eligible auth candidate for this key", http.StatusServiceUnavailable), nil
	}

	idx := int(atomic.AddUint64(&roundRobin, 1)-1) % len(filtered)
	return OKEnvelope(SchedulerPickResponse{AuthID: filtered[idx].ID, Handled: true})
}

// extractAPIKey reads the caller api-key from the same headers the built-in
// config-api-key access provider accepts.
func extractAPIKey(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}
	if token := bearerToken(firstHeader(headers, "Authorization")); token != "" {
		return token
	}
	for _, name := range []string{"X-Api-Key", "X-Goog-Api-Key"} {
		if v := firstHeader(headers, name); v != "" {
			return v
		}
	}
	return ""
}

func firstHeader(headers map[string][]string, names ...string) string {
	for _, name := range names {
		for key, values := range headers {
			if !strings.EqualFold(key, name) {
				continue
			}
			for _, value := range values {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

// bearerToken mirrors internal/access/config_access/provider.go extractBearerToken
// so the value matched here is identical to what the access layer validated.
func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return header
}

// candidateUsable mirrors the host's notion of an unusable auth status.
func candidateUsable(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.NewReplacer("-", "_", " ", "_").Replace(status)
	switch status {
	case "disabled", "error", "expired", "revoked", "invalid", "unavailable",
		"cooldown", "cooling_down", "quota_exhausted", "exhausted", "blocked":
		return false
	default:
		return true
	}
}

// candidateAllowed reports whether a candidate matches any allow entry:
//   - "auth:<id>" matches the candidate account ID exactly;
//   - any other entry matches candidate.Provider (case-insensitive), e.g.
//     "claude", "gemini", "openrouter".
func candidateAllowed(allow []string, cand SchedulerAuthCandidate) bool {
	provider := strings.ToLower(strings.TrimSpace(cand.Provider))
	for _, entry := range allow {
		entry = strings.TrimSpace(entry)
		switch {
		case entry == "":
			continue
		case strings.HasPrefix(entry, "auth:"):
			if target := strings.TrimSpace(strings.TrimPrefix(entry, "auth:")); target == cand.ID {
				return true
			}
		default:
			if strings.ToLower(entry) == provider {
				return true
			}
		}
	}
	return false
}
