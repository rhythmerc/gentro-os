package games

import (
	"context"
	"maps"
	"os/exec"
	"slices"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// GameSource defines the interface for game sources (Steam, emulated, etc.)
type GameSource interface {
	// Name returns the source identifier (e.g., "steam", "emulated")
	Name() string

	// Init initializes the source with configuration
	Init(config map[string]any) error

	// GetInstances returns all game instances from this source
	// For Steam: returns installed games (Web API for library games planned)
	// For emulated: returns all discovered ROMs
	GetInstances(ctx context.Context) ([]models.GameInstance, error)

	// GetGameArt returns art data for a specific game
	// Returns: (data []byte, contentType string, error)
	GetGameArt(ctx context.Context, instanceID string, artType string) ([]byte, string, error)

	// Refresh updates the source's internal cache/state
	Refresh(ctx context.Context) error

	// Launch initiates the game and returns the running command
	// Returns (*exec.Cmd, error) where cmd.Process is the started process
	Launch(ctx context.Context, instance models.GameInstance) (*exec.Cmd, error)

	// MonitorProcess watches the game process and emits status events
	// Source-specific implementation:
	// - Emulated: Wait() for direct process exit
	// - Steam: Activity-based polling with threshold
	MonitorProcess(ctx context.Context, instance models.GameInstance, cmd *exec.Cmd)

	// FilterInstances applies source-specific filters to a batch of instances
	// Each source handles its own filtering logic (e.g., Steam tools filtering)
	FilterInstances(instances []models.GameInstance, filter models.GameFilter) []models.GameInstance
}

// SourceRegistry manages multiple game sources
type SourceRegistry struct {
	sources map[string]GameSource
}

// NewSourceRegistry creates a new source registry
func NewSourceRegistry() *SourceRegistry {
	return &SourceRegistry{
		sources: make(map[string]GameSource),
	}
}

// Register adds a source to the registry
func (r *SourceRegistry) Register(source GameSource) error {
	if err := source.Init(nil); err != nil {
		return err
	}
	r.sources[source.Name()] = source
	return nil
}

// RegisterWithConfig adds a source with configuration
func (r *SourceRegistry) RegisterWithConfig(source GameSource, config map[string]any) error {
	if err := source.Init(config); err != nil {
		return err
	}
	r.sources[source.Name()] = source
	return nil
}

// Get returns a source by name
func (r *SourceRegistry) Get(name string) (GameSource, bool) {
	source, ok := r.sources[name]
	return source, ok
}

// GetAll returns all registered sources
func (r *SourceRegistry) GetAll() []GameSource {
	return slices.Collect(maps.Values(r.sources))
}

// GetNames returns all registered source names
func (r *SourceRegistry) GetNames() []string {
	return slices.Collect(maps.Keys(r.sources))
}
