package games

import (
	"context"
	"os/exec"
	"testing"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// MockSource implements GameSource for testing
type MockSource struct {
	name      string
	instances []models.GameInstance
}

func (m *MockSource) Name() string                     { return m.name }
func (m *MockSource) Init(config map[string]any) error { return nil }
func (m *MockSource) GetInstances(ctx context.Context) ([]models.GameInstance, error) {
	return m.instances, nil
}
func (m *MockSource) GetGameArt(ctx context.Context, instanceID string, artType string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *MockSource) Refresh(ctx context.Context) error { return nil }
func (m *MockSource) Launch(ctx context.Context, instance models.GameInstance) (*exec.Cmd, error) {
	return nil, nil
}
func (m *MockSource) MonitorProcess(ctx context.Context, instance models.GameInstance, cmd *exec.Cmd) {
}

func (m *MockSource) FilterInstances(instances []models.GameInstance, filter models.GameFilter) []models.GameInstance {
	return instances
}

func TestApplySourceFilters(t *testing.T) {
	// Create a service with a mock registry
	service := &GamesService{
		registry: NewSourceRegistry(),
	}

	// Register a mock source
	mockSource := &MockSource{name: "mock"}
	service.registry.Register(mockSource)

	// Create test instances
	instances := []models.GameInstance{
		{ID: "1", Source: "mock", GameID: "game1"},
		{ID: "2", Source: "mock", GameID: "game2"},
	}

	// Apply filters
	filter := models.GameFilter{}
	result := service.applySourceFilters(instances, filter)

	// Should return all instances (no filtering in mock)
	if len(result) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(result))
	}
}

func TestGetDefaultFilterConfig(t *testing.T) {
	service := &GamesService{}

	// Test without config manager (should use hardcoded defaults)
	filter := service.GetDefaultFilterConfig()

	if filter.SourceFilters == nil {
		t.Fatal("SourceFilters should not be nil")
	}

	steamFilters, ok := filter.SourceFilters["steam"]
	if !ok {
		t.Fatal("Steam filters not found")
	}

	excludeTools, ok := steamFilters["excludeTools"].(bool)
	if !ok {
		t.Fatal("excludeTools not found or not bool")
	}

	if !excludeTools {
		t.Error("Expected excludeTools to default to true")
	}
}
