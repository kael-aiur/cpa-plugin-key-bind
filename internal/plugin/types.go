package plugin

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// ABIVersion / SchemaVersion track the native C ABI / RPC JSON contract. They
// must match the host (CLIProxyAPI sdk/pluginabi).
const (
	ABIVersion    uint32 = 1
	SchemaVersion uint32 = 1

	MethodPluginRegister     = "plugin.register"
	MethodPluginReconfigure  = "plugin.reconfigure"
	MethodSchedulerPick      = "scheduler.pick"
	MethodManagementRegister = "management.register"
	MethodManagementHandle   = "management.handle"
)

const (
	PluginID   = "key-bind"
	PluginName = "key-bind"
	Version    = "0.1.1"
)

// --- RPC envelope ---

type Envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *EnvelopeError  `json:"error,omitempty"`
}

type EnvelopeError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

type LifecycleRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

// --- Registration ---

type Registration struct {
	SchemaVersion uint32       `json:"schema_version"`
	Metadata      Metadata     `json:"metadata"`
	Capabilities  Capabilities `json:"capabilities"`
}

type Metadata struct {
	Name             string        `json:"Name"`
	Version          string        `json:"Version"`
	Author           string        `json:"Author"`
	GitHubRepository string        `json:"GitHubRepository,omitempty"`
	Logo             string        `json:"Logo,omitempty"`
	ConfigFields     []ConfigField `json:"ConfigFields"`
}

type ConfigField struct {
	Name        string   `json:"Name"`
	Type        string   `json:"Type"`
	EnumValues  []string `json:"EnumValues,omitempty"`
	Description string   `json:"Description"`
}

// Capabilities declares only the host hooks this plugin implements.
// key-bind uses scheduler.pick (filter candidates by key) + management API
// (CRUD bindings + serve the config UI).
type Capabilities struct {
	Scheduler     bool `json:"scheduler,omitempty"`
	ManagementAPI bool `json:"management_api"`
}

// --- scheduler.pick ---

// SchedulerPickRequest mirrors pluginapi.SchedulerPickRequest. The plugin uses
// Options.Headers (to read the caller's api-key) and Candidates (to filter).
type SchedulerPickRequest struct {
	Provider   string                   `json:"Provider,omitempty"`
	Providers  []string                 `json:"Providers,omitempty"`
	Model      string                   `json:"Model"`
	Stream     bool                     `json:"Stream,omitempty"`
	Options    SchedulerPickOptions     `json:"Options"`
	Candidates []SchedulerAuthCandidate `json:"Candidates"`
}

type SchedulerPickOptions struct {
	Headers  map[string][]string `json:"Headers,omitempty"`
	Metadata map[string]any      `json:"Metadata,omitempty"`
}

// SchedulerAuthCandidate describes one selectable auth record.
type SchedulerAuthCandidate struct {
	ID         string            `json:"ID"`
	Provider   string            `json:"Provider"`
	Priority   int               `json:"Priority,omitempty"`
	Status     string            `json:"Status,omitempty"`
	Attributes map[string]string `json:"Attributes,omitempty"`
	Metadata   map[string]any    `json:"Metadata,omitempty"`
}

type SchedulerPickResponse struct {
	// AuthID picks a specific candidate; leave empty with Handled=false to defer
	// to the host scheduler (i.e. "no binding for this key -> platform default").
	AuthID string `json:"AuthID,omitempty"`
	// Handled reports whether the plugin made a scheduling decision.
	Handled bool `json:"Handled"`
}

// --- management.register / management.handle ---

type ManagementRegistrationResponse struct {
	Routes    []ManagementRoute `json:"Routes"`
	Resources []ResourceRoute   `json:"Resources,omitempty"`
}

type ManagementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Description string `json:"Description,omitempty"`
}

// ResourceRoute declares a browser-navigable, unauthenticated GET resource that
// the host serves under /v0/resource/plugins/<pluginID><Path>. Menu is the label
// shown in the management UI's plugin entry.
type ResourceRoute struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu,omitempty"`
	Description string `json:"Description,omitempty"`
}

type ManagementRequest struct {
	Method  string      `json:"Method"`
	Path    string      `json:"Path"`
	Headers http.Header `json:"Headers"`
	Query   url.Values  `json:"Query"`
	Body    []byte      `json:"Body"`
}

type ManagementResponse struct {
	StatusCode int         `json:"StatusCode,omitempty"`
	Headers    http.Header `json:"Headers,omitempty"`
	Body       []byte      `json:"Body,omitempty"`
}

// --- envelope helpers ---

func OKEnvelope(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{OK: true, Result: raw})
}

func ErrorEnvelope(code, message string, status int) []byte {
	raw, _ := json.Marshal(Envelope{
		OK: false,
		Error: &EnvelopeError{
			Code:       code,
			Message:    message,
			HTTPStatus: status,
		},
	})
	return raw
}
