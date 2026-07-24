package plugin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"cpa-plugin-key-bind/internal/plugin/web"
)

const (
	resourceBase = "/v0/resource/plugins/" + PluginID
	mgmtBase     = "/v0/management/plugins/" + PluginID
)

// managementRegistration declares the management API routes and the web UI
// resource exposed by this plugin.
func (a *App) managementRegistration() ManagementRegistrationResponse {
	return ManagementRegistrationResponse{
		Routes: []ManagementRoute{
			{Method: http.MethodGet, Path: mgmtBase + "/binds", Description: "List key→provider bindings."},
			{Method: http.MethodPost, Path: mgmtBase + "/binds", Description: "Create a key→provider binding."},
			{Method: http.MethodPut, Path: mgmtBase + "/binds", Description: "Update a key→provider binding by id."},
			{Method: http.MethodDelete, Path: mgmtBase + "/binds", Description: "Delete a key→provider binding by id."},
		},
		Resources: []ResourceRoute{
			{Path: web.IndexPath, Menu: "Key Bind", Description: "Web UI for managing key→provider bindings."},
		},
	}
}

// handleManagement dispatches a management API request: CRUD on /binds, plus the
// unauthenticated browser UI served under /v0/resource/plugins/key-bind.
func (a *App) handleManagement(raw []byte) ([]byte, error) {
	var req ManagementRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	path := strings.TrimRight(req.Path, "/")

	// Plugin resource GETs (unauthenticated browser UI) go through the same
	// management.handle dispatch.
	if req.Method == http.MethodGet && strings.HasPrefix(path, resourceBase) {
		status, headers, body := web.Serve(strings.TrimPrefix(path, resourceBase))
		return OKEnvelope(ManagementResponse{StatusCode: status, Headers: headers, Body: body})
	}

	switch {
	case req.Method == http.MethodGet && path == mgmtBase+"/binds":
		return OKEnvelope(jsonResponse(http.StatusOK, map[string]any{"bindings": a.store.List()}))
	case req.Method == http.MethodPost && path == mgmtBase+"/binds":
		return OKEnvelope(a.createBinding(req.Body))
	case req.Method == http.MethodPut && path == mgmtBase+"/binds":
		return OKEnvelope(a.updateBinding(req.Body))
	case req.Method == http.MethodDelete && path == mgmtBase+"/binds":
		return OKEnvelope(a.deleteBinding(req.Query, req.Body))
	default:
		return OKEnvelope(jsonError(http.StatusNotFound, "not_found", "unknown management route"))
	}
}

// bindingInput is the create/update request body. Key is the plaintext api-key
// (hashed on store); omit on update to keep the existing key.
type bindingInput struct {
	ID      string   `json:"id,omitempty"`
	Name    string   `json:"name,omitempty"`
	Key     string   `json:"key,omitempty"`
	Allow   []string `json:"allow,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}

func (a *App) createBinding(body []byte) ManagementResponse {
	var in bindingInput
	if err := json.Unmarshal(body, &in); err != nil {
		return jsonError(http.StatusBadRequest, "invalid_request", err.Error())
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	b, err := a.store.Create(in.Name, in.Key, in.Allow, enabled)
	if err != nil {
		return jsonError(http.StatusBadRequest, "create_failed", err.Error())
	}
	return jsonResponse(http.StatusCreated, b)
}

func (a *App) updateBinding(body []byte) ManagementResponse {
	var in bindingInput
	if err := json.Unmarshal(body, &in); err != nil {
		return jsonError(http.StatusBadRequest, "invalid_request", err.Error())
	}
	if strings.TrimSpace(in.ID) == "" {
		return jsonError(http.StatusBadRequest, "invalid_request", "id is required")
	}
	b, err := a.store.Update(in.ID, in.Name, in.Key, in.Allow, in.Enabled)
	if err != nil {
		return jsonError(http.StatusNotFound, "not_found", err.Error())
	}
	return jsonResponse(http.StatusOK, b)
}

func (a *App) deleteBinding(query url.Values, body []byte) ManagementResponse {
	id := strings.TrimSpace(query.Get("id"))
	if id == "" {
		var in bindingInput
		_ = json.Unmarshal(body, &in)
		id = strings.TrimSpace(in.ID)
	}
	if id == "" {
		return jsonError(http.StatusBadRequest, "invalid_request", "id is required")
	}
	if err := a.store.Delete(id); err != nil {
		return jsonError(http.StatusNotFound, "not_found", err.Error())
	}
	return jsonResponse(http.StatusOK, map[string]any{"id": id, "deleted": true})
}

func jsonResponse(status int, v any) ManagementResponse {
	body, _ := json.Marshal(v)
	return ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       body,
	}
}

func jsonError(status int, code, message string) ManagementResponse {
	return jsonResponse(status, map[string]any{"error": code, "message": message})
}
