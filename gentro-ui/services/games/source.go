package games

import (
	"context"
	"os/exec"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// GameSource defines the interface for game sources (Steam, file-based, etc.)
type GameSource interface {
	// Name returns the source identifier (e.g., "steam", "file")
	Name() string

	// Init initializes the source with configuration
	Init(config map[string]any) error

	// GetInstances returns all game instances from this source
	// For Steam: returns installed games + library games
	// For file: returns all discovered ROMs
	GetInstances(ctx context.Context) ([]models.GameInstance, error)

	// GetInstalledInstances returns only locally installed/running games
	GetInstalledInstances(ctx context.Context) ([]models.GameInstance, error)

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
	sources := make([]GameSource, 0, len(r.sources))
	for _, source := range r.sources {
		sources = append(sources, source)
	}
	return sources
}

// GetNames returns all registered source names
func (r *SourceRegistry) GetNames() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}
