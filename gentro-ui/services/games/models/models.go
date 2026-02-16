package models

import (
	"time"
)

// MetadataState represents the state of metadata fetching
type MetadataState string

const (
	MetadataStateIdle      MetadataState = "idle"
	MetadataStateFetching  MetadataState = "fetching"
	MetadataStateCompleted MetadataState = "completed"
	MetadataStateError     MetadataState = "error"
	MetadataStateCancelled MetadataState = "cancelled"
)

// Game represents the abstract game entity
type Game struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	ReleaseDate *time.Time        `json:"releaseDate,omitempty" db:"release_date"`
	Developer   string            `json:"developer" db:"developer"`
	Publisher   string            `json:"publisher" db:"publisher"`
	Genres      []string          `json:"genres" db:"-"`
	Platforms   []string          `json:"platforms" db:"-"`
	ArtURLs     map[string]string `json:"artUrls" db:"-"`
	CreatedAt   time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time         `json:"updatedAt" db:"updated_at"`
}

// GameInstance represents a specific copy/installation of a game
type GameInstance struct {
	ID             string         `json:"id" db:"id"`
	GameID         string         `json:"gameId" db:"game_id"`
	Source         string         `json:"source" db:"source"`
	Platform       string         `json:"platform" db:"platform"`
	SourceID       string         `json:"sourceId" db:"source_id"`
	Path           string         `json:"path,omitempty" db:"path"`
	Filename       string         `json:"filename,omitempty" db:"filename"`
	FileSize       int64          `json:"fileSize,omitempty" db:"file_size"`
	FileHash       string         `json:"fileHash,omitempty" db:"file_hash"`
	Installed      bool           `json:"installed" db:"installed"`
	InstallPath    string         `json:"installPath,omitempty" db:"install_path"`
	MetadataStatus MetadataStatus `json:"metadataStatus" db:"-"`
	CustomMetadata map[string]any `json:"customMetadata" db:"-"`
	SourceData     map[string]any `json:"sourceData,omitempty" db:"-"`
	CreatedAt      time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time      `json:"updatedAt" db:"updated_at"`
}

// MetadataStatus tracks async metadata fetching progress
type MetadataStatus struct {
	State        MetadataState `json:"state"`
	Progress     float64       `json:"progress"`
	Message      string        `json:"message,omitempty"`
	Error        string        `json:"error,omitempty"`
	StartedAt    *time.Time    `json:"startedAt,omitempty"`
	CompletedAt  *time.Time    `json:"completedAt,omitempty"`
	SourcesTried []string      `json:"sourcesTried,omitempty"`
}

// MetadataLayer represents the fallback hierarchy
type MetadataLayer struct {
	External         map[string]any            `json:"external,omitempty"`
	GameCustom       map[string]any            `json:"gameCustom,omitempty"`
	PlatformMetadata map[string]map[string]any `json:"platformMetadata,omitempty"`
	InstanceCustom   map[string]any            `json:"instanceCustom,omitempty"`
}

// GameFilter represents filtering options for games
type GameFilter struct {
	InstalledOnly bool     `json:"installedOnly"`
	Source        string   `json:"source,omitempty"`
	Platform      string   `json:"platform,omitempty"`
	Search        string   `json:"search,omitempty"`
	Genres        []string `json:"genres,omitempty"`

	// SourceFilters allows source-specific filtering
	// Key is source name (e.g., "steam"), value is map of filter options
	SourceFilters map[string]map[string]any `json:"sourceFilters,omitempty"`
}

// GameSort represents sorting options for games
type GameSort struct {
	Field string `json:"field"` // "name", "lastPlayed", "fileSize", "dateAdded"
	Order string `json:"order"` // "asc", "desc"
}

// Sort field constants
const (
	SortByName       = "name"
	SortByLastPlayed = "lastPlayed"
	SortByFileSize   = "fileSize"
	SortByDateAdded  = "dateAdded"

	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// GameWithInstance combines game and instance data for UI
type GameWithInstance struct {
	Game     Game         `json:"game"`
	Instance GameInstance `json:"instance"`
}

// FetchRequest represents a metadata fetch request
type FetchRequest struct {
	GameID     string
	InstanceID string
	Priority   int
	Platforms  []string
	Name       string
	FileHash   string
}

// ResolvedMetadata contains metadata from external sources
type ResolvedMetadata struct {
	GameMetadata     GameMetadata
	PlatformMetadata map[string]PlatformMetadata
	ArtURLs          map[string]string
}

// GameMetadata represents game-level metadata from external sources
type GameMetadata struct {
	Name        string
	Description string
	ReleaseDate *time.Time
	Developer   string
	Publisher   string
	Genres      []string
}

// PlatformMetadata represents platform-specific metadata
type PlatformMetadata struct {
	Platform    string
	ReleaseDate *time.Time
	Region      string
	Rating      string
}

// MetadataStatusUpdate is sent via Wails events
type MetadataStatusUpdate struct {
	InstanceID string         `json:"instanceId"`
	GameID     string         `json:"gameId"`
	Status     MetadataStatus `json:"status"`
}

// LaunchStatus represents the state of game launching/running
type LaunchStatus string

const (
	LaunchStatusLaunching LaunchStatus = "launching"
	LaunchStatusRunning   LaunchStatus = "running"
	LaunchStatusStopped   LaunchStatus = "stopped"
	LaunchStatusFailed    LaunchStatus = "failed"
)

// LaunchStatusUpdate is sent via Wails events when game launch status changes
type LaunchStatusUpdate struct {
	InstanceID string       `json:"instanceId"`
	GameID     string       `json:"gameId"`
	Status     LaunchStatus `json:"status"`
	Error      string       `json:"error,omitempty"`
}

// EmulatorType represents how the emulator is installed
type EmulatorType string

const (
	EmulatorTypeFlatpak  EmulatorType = "flatpak"
	EmulatorTypeNative   EmulatorType = "native"
	EmulatorTypeAppImage EmulatorType = "appimage"
)

// Emulator represents an emulator configuration
type Emulator struct {
	ID                 string       `json:"id" db:"id"`
	Name               string       `json:"name" db:"name"`
	DisplayName        string       `json:"displayName" db:"display_name"`
	Type               EmulatorType `json:"type" db:"type"`
	ExecutablePath     string       `json:"executablePath,omitempty" db:"executable_path"`
	FlatpakID          string       `json:"flatpakId,omitempty" db:"flatpak_id"`
	CommandTemplate    string       `json:"commandTemplate" db:"command_template"`
	DefaultArgs        string       `json:"defaultArgs,omitempty" db:"default_args"`
	SupportedPlatforms []string     `json:"supportedPlatforms" db:"supported_platforms"`
	IsAvailable        bool         `json:"isAvailable" db:"is_available"`
	CreatedAt          time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time    `json:"updatedAt" db:"updated_at"`
}

// EmulatorCore represents a RetroArch core (Option B)
type EmulatorCore struct {
	ID                 string   `json:"id" db:"id"`
	EmulatorID         string   `json:"emulatorId" db:"emulator_id"`
	CoreID             string   `json:"coreId" db:"core_id"`
	DisplayName        string   `json:"displayName" db:"display_name"`
	SupportedPlatforms []string `json:"supportedPlatforms" db:"supported_platforms"`
	IsAvailable        bool     `json:"isAvailable" db:"is_available"`
}

// PlatformEmulator maps platforms to available emulators/cores
type PlatformEmulator struct {
	ID         string `json:"id" db:"id"`
	Platform   string `json:"platform" db:"platform"`
	EmulatorID string `json:"emulatorId" db:"emulator_id"`
	CoreID     string `json:"coreId,omitempty" db:"core_id"`
	IsDefault  bool   `json:"isDefault" db:"is_default"`
}

// InstanceEmulatorSettings for per-game overrides
type InstanceEmulatorSettings struct {
	InstanceID string `json:"instanceId" db:"instance_id"`
	EmulatorID string `json:"emulatorId" db:"emulator_id"`
	CoreID     string `json:"coreId,omitempty" db:"core_id"`
	CustomArgs string `json:"customArgs,omitempty" db:"custom_args"`
}
