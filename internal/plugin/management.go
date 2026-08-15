package plugin

import (
	"encoding/json"
	"net/http"
	"strings"

	"cpa-plugin-key-bind/internal/plugin/web"
)

const (
	resourceBase = "/v0/resource/plugins/" + PluginID
	mgmtBase     = "/v0/management/plugins/" + PluginID
)

// managementRegistration declares only the browser UI resource exposed by this
// plugin. Binding CRUD is handled by the host plugin-config API.
func (a *App) managementRegistration() ManagementRegistrationResponse {
	return ManagementRegistrationResponse{
		Routes: []ManagementRoute{},
		Resources: []ResourceRoute{
			{Path: web.IndexPath, Menu: "密钥绑定", Description: "绑定密钥和供应商账号"},
		},
	}
}

// handleManagement dispatches management requests for the unauthenticated
// browser UI. Binding CRUD routes are intentionally not served here.
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

	return OKEnvelope(jsonError(http.StatusNotFound, "not_found", "unknown management route"))
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
