package plugin

import (
	"encoding/json"
	"net/http"

	"gopkg.in/yaml.v3"

	"cpa-plugin-key-bind/internal/store"
)

// App holds plugin runtime state. The only stateful component is the binding
// store; scheduler picks are stateless lookups against it.
type App struct {
	store *store.Store
}

// NewApp constructs an App with an empty store. The store is (re)loaded from the
// state file on plugin.register / plugin.reconfigure.
func NewApp() *App {
	return &App{store: store.NewStore()}
}

// Shutdown is a no-op (no background workers to stop).
func (a *App) Shutdown() {}

// HandleMethod is the single entry point dispatched by the C-ABI call handler.
func (a *App) HandleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case MethodPluginRegister, MethodPluginReconfigure:
		if err := a.configure(request); err != nil {
			return nil, err
		}
		return OKEnvelope(a.registration())
	case MethodSchedulerPick:
		return a.pickAuth(request)
	case MethodManagementRegister:
		return OKEnvelope(a.managementRegistration())
	case MethodManagementHandle:
		return a.handleManagement(request)
	default:
		return ErrorEnvelope("unknown_method", "unknown method: "+method, http.StatusBadRequest), nil
	}
}

// pluginConfig mirrors the plugins.configs.key-bind YAML block.
type pluginConfig struct {
	StateFile string `yaml:"state_file" json:"state_file"`
}

// configure parses the config_yaml payload (state_file) and (re)loads the store.
func (a *App) configure(raw []byte) error {
	cfg := pluginConfig{}
	if len(raw) > 0 {
		var req LifecycleRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return err
		}
		if len(req.ConfigYAML) > 0 {
			if err := yaml.Unmarshal(req.ConfigYAML, &cfg); err != nil {
				return err
			}
		}
	}
	statePath, err := store.ResolveStatePath(cfg.StateFile)
	if err != nil {
		return err
	}
	return a.store.Configure(statePath)
}

// registration declares plugin metadata and capabilities to the host.
func (a *App) registration() Registration {
	return Registration{
		SchemaVersion: SchemaVersion,
		Metadata: Metadata{
			Name:    PluginName,
			Version: Version,
			Author:  "kael",
			ConfigFields: []ConfigField{
				{
					Name:        "state_file",
					Type:        "string",
					Description: "JSON state file storing key→provider bindings (edited via the plugin UI).",
				},
			},
		},
		Capabilities: Capabilities{Scheduler: true, ManagementAPI: true},
	}
}
