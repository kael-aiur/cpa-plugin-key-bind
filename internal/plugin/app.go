package plugin

import (
	"encoding/json"
	"net/http"
	"sync"

	"gopkg.in/yaml.v3"

	"cpa-plugin-key-bind/internal/bindings"
	"cpa-plugin-key-bind/internal/store"
)

// App holds the immutable host-config snapshot used by the scheduler. The
// legacy store remains available to the management routes until their separate
// migration removes JSON persistence.
type App struct {
	mu    sync.RWMutex
	index *bindings.Index
	store *store.Store
}

// NewApp constructs an App with an empty host-config index.
func NewApp() *App {
	return &App{index: bindings.Empty(), store: store.NewStore()}
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
	Bindings []bindings.ConfigBinding `yaml:"bindings" json:"bindings"`
}

// configure parses the host-supplied config_yaml and atomically publishes a new
// immutable bindings index. Failed builds leave the previous snapshot active.
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

	next, err := bindings.Build(cfg.Bindings)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.index = next
	a.mu.Unlock()
	return nil
}

func (a *App) activeBindings() *bindings.Index {
	a.mu.RLock()
	index := a.index
	a.mu.RUnlock()
	return index
}

// registration declares plugin metadata and capabilities to the host.
func (a *App) registration() Registration {
	return Registration{
		SchemaVersion: SchemaVersion,
		Metadata: Metadata{
			Name:             PluginName,
			Version:          Version,
			Author:           "kael",
			GitHubRepository: "https://github.com/kael-aiur/cpa-plugin-key-bind",
			ConfigFields: []ConfigField{
				{
					Name:        "bindings",
					Type:        "array",
					Description: "Key hashes and their allowed providers/accounts. Managed by the key-bind UI.",
				},
			},
		},
		Capabilities: Capabilities{Scheduler: true, ManagementAPI: true},
	}
}
