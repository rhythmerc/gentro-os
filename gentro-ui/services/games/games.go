package games

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/rhythmerc/gentro-ui/services/config"
	"github.com/rhythmerc/gentro-ui/services/games/database"
	"github.com/rhythmerc/gentro-ui/services/games/emulator"
	"github.com/rhythmerc/gentro-ui/services/games/metadata"
	"github.com/rhythmerc/gentro-ui/services/games/models"
	"github.com/rhythmerc/gentro-ui/services/games/sources/emulated"
	"github.com/rhythmerc/gentro-ui/services/games/sources/steam"
)

// GamesService manages games from multiple sources
type GamesService struct {
	db          *database.DB
	registry    *SourceRegistry
	fetcher     *metadata.Fetcher
	emuService  *emulator.Service
	config      *config.Manager
	route       string
	logger      *slog.Logger
	artCacheDir string
}

// GamesServiceConfig holds service configuration
type GamesServiceConfig struct {
	DatabasePath string
	ArtCacheDir  string
	Logger       *slog.Logger
}

// NewGamesService creates a new GamesService
func NewGamesService(config GamesServiceConfig) (*GamesService, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	// Set default paths
	if config.DatabasePath == "" {
		home := os.Getenv("HOME")
		config.DatabasePath = filepath.Join(home, ".local", "share", "gentro", "database", "games.db")
	}
	if config.ArtCacheDir == "" {
		home := os.Getenv("HOME")
		config.ArtCacheDir = filepath.Join(home, ".local", "share", "gentro", "cache", "art")
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(config.DatabasePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	if err := os.MkdirAll(config.ArtCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create art cache directory: %w", err)
	}

	// Initialize database
	db, err := database.New(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize source registry
	registry := NewSourceRegistry()

	// Initialize metadata fetcher
	fetcher := metadata.NewFetcher(2, config.Logger)
	fetcher.RegisterResolver(&metadata.LocalCacheResolver{})

	// Initialize emulator service
	emuService := emulator.NewService(db, config.Logger)

	service := &GamesService{
		db:          db,
		registry:    registry,
		fetcher:     fetcher,
		emuService:  emuService,
		logger:      config.Logger,
		artCacheDir: config.ArtCacheDir,
	}

	return service, nil
}

// ServiceStartup runs when the app starts
func (s *GamesService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	// Set default route
	s.route = "/games"

	// Initialize config manager
	configPath := config.DefaultConfigPath()
	s.logger.Info("Initializing config manager", "path", configPath)
	cfgManager, err := config.NewManager(configPath)
	if err != nil {
		s.logger.Error("failed to initialize config manager", "error", err)
		// Continue without config - we'll use defaults
	} else {
		s.config = cfgManager
	}

	// Initialize emulators (seed defaults)
	s.logger.Info("Initializing emulators")
	if err := s.emuService.Initialize(); err != nil {
		s.logger.Error("failed to initialize emulators", "error", err)
	}

	// Discover available emulators
	s.logger.Info("Discovering available emulators")
	if err := s.emuService.DiscoverAvailable(); err != nil {
		s.logger.Error("failed to discover emulators", "error", err)
	}

	// Register default sources
	emulatedSource := &emulated.Source{}
	if err := s.registry.Register(emulatedSource); err != nil {
		s.logger.Warn("failed to register emulated source", "error", err)
	} else {
		// Inject emulator service and logger into emulated source
		emulatedSource.SetEmulatorService(s.emuService)
		emulatedSource.SetLogger(s.logger)
	}

	if err := s.registry.Register(&steam.Source{}); err != nil {
		s.logger.Warn("failed to register steam source", "error", err)
	}

	// Start metadata fetcher
	s.fetcher.Start()

	// Initial sync
	go s.RefreshGames()

	return nil
}

// ServiceShutdown runs when the app shuts down
func (s *GamesService) ServiceShutdown(ctx context.Context) error {
	s.fetcher.Stop()
	return s.db.Close()
}

// GetGames returns games with optional filtering and sorting
func (s *GamesService) GetGames(filter *models.GameFilter, sortOpts *models.GameSort) ([]models.GameWithInstance, error) {
	// Apply defaults if nil
	effectiveFilter := filter
	if effectiveFilter == nil {
		effectiveFilter = &models.GameFilter{
			InstalledOnly: false,
			SourceFilters: map[string]map[string]any{
				"steam": {"excludeTools": true},
			},
		}
	}

	effectiveSort := sortOpts
	if effectiveSort == nil {
		effectiveSort = &models.GameSort{
			Field: models.SortByName,
			Order: models.SortOrderAsc,
		}
	}

	// Get instances from database
	instances, err := s.db.GetInstances(*effectiveFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	// Apply source-specific filters
	instances = s.applySourceFilters(instances, *effectiveFilter)

	// Build game map to avoid duplicates
	gameMap := make(map[string]*models.Game)
	var result []models.GameWithInstance

	for _, instance := range instances {
		// Get or load game
		game, ok := gameMap[instance.GameID]
		if !ok {
			game, err = s.db.GetGame(instance.GameID)
			if err != nil {
				s.logger.Warn("failed to load game", "gameID", instance.GameID, "error", err)
				continue
			}
			if game == nil {
				// Create placeholder game if not found
				game = &models.Game{
					ID:        instance.GameID,
					Name:      s.getDisplayName(instance),
					Platforms: []string{instance.Platform},
				}
				if err := s.db.CreateGame(game); err != nil {
					s.logger.Warn("failed to create placeholder game", "error", err)
				}
			}
			gameMap[instance.GameID] = game
		}

		// Apply search filter
		if effectiveFilter.Search != "" && !strings.Contains(strings.ToLower(game.Name), strings.ToLower(effectiveFilter.Search)) {
			continue
		}

		// Apply genre filter
		if len(effectiveFilter.Genres) > 0 {
			// Check if game has any of the specified genres
			hasGenre := false
			for _, filterGenre := range effectiveFilter.Genres {
				for _, gameGenre := range game.Genres {
					if strings.EqualFold(gameGenre, filterGenre) {
						hasGenre = true
						break
					}
				}
			}
			if !hasGenre {
				continue
			}
		}

		result = append(result, models.GameWithInstance{
			Game:     *game,
			Instance: instance,
		})
	}

	// Apply sorting
	result = s.sortGames(result, effectiveSort)

	return result, nil
}

// sortGames sorts games by the specified field and order
func (s *GamesService) sortGames(games []models.GameWithInstance, sortOpts *models.GameSort) []models.GameWithInstance {
	if sortOpts == nil || sortOpts.Field == "" {
		return games
	}

	sort.Slice(games, func(i, j int) bool {
		var cmp int

		switch sortOpts.Field {
		case models.SortByName:
			cmp = strings.Compare(strings.ToLower(games[i].Game.Name), strings.ToLower(games[j].Game.Name))
		case models.SortByFileSize:
			cmp = int(games[i].Instance.FileSize - games[j].Instance.FileSize)
		case models.SortByDateAdded:
			cmp = games[i].Instance.CreatedAt.Compare(games[j].Instance.CreatedAt)
		default:
			cmp = strings.Compare(strings.ToLower(games[i].Game.Name), strings.ToLower(games[j].Game.Name))
		}

		if sortOpts.Order == models.SortOrderDesc {
			return cmp > 0
		}
		return cmp < 0
	})

	return games
}

// applySourceFilters groups instances by source and applies source-specific filters
func (s *GamesService) applySourceFilters(instances []models.GameInstance, filter models.GameFilter) []models.GameInstance {
	if len(instances) == 0 {
		return instances
	}

	// Group instances by source
	instancesBySource := make(map[string][]models.GameInstance)
	for _, instance := range instances {
		instancesBySource[instance.Source] = append(instancesBySource[instance.Source], instance)
	}

	// Apply source-specific filters
	var filteredInstances []models.GameInstance
	for sourceName, sourceInstances := range instancesBySource {
		source, ok := s.registry.Get(sourceName)
		if !ok {
			s.logger.Warn("source not found for filtering", "source", sourceName)
			// Include instances from unknown sources (fail open)
			filteredInstances = append(filteredInstances, sourceInstances...)
			continue
		}

		// Apply source-specific filter
		sourceFiltered := source.FilterInstances(sourceInstances, filter)
		filteredInstances = append(filteredInstances, sourceFiltered...)
	}

	return filteredInstances
}

// GetDefaultFilterConfig returns the default filter configuration from config
func (s *GamesService) GetDefaultFilterConfig() models.GameFilter {
	filter := models.GameFilter{
		InstalledOnly: false,
		SourceFilters: make(map[string]map[string]any),
	}

	// Add Steam filter defaults
	if s.config != nil {
		cfg := s.config.Get()
		filter.SourceFilters["steam"] = map[string]any{
			"excludeTools": cfg.Filters.Steam.ExcludeTools,
		}
	} else {
		// Fallback to hardcoded defaults
		filter.SourceFilters["steam"] = map[string]any{
			"excludeTools": true,
		}
	}

	return filter
}

// UpdateFilterConfig updates the filter configuration
func (s *GamesService) UpdateFilterConfig(steamExcludeTools bool) error {
	if s.config == nil {
		return fmt.Errorf("config manager not initialized")
	}

	newFilters := config.FilterConfig{
		Steam: config.SteamFilterConfig{
			ExcludeTools: steamExcludeTools,
		},
	}

	return s.config.SetFilters(newFilters)
}

// GetGame returns a single game with all its instances
func (s *GamesService) GetGame(gameID string) (*models.Game, []models.GameInstance, error) {
	game, err := s.db.GetGame(gameID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get game: %w", err)
	}
	if game == nil {
		return nil, nil, fmt.Errorf("game not found: %s", gameID)
	}

	// Get all instances for this game
	filter := models.GameFilter{}
	allInstances, err := s.db.GetInstances(filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get instances: %w", err)
	}

	var instances []models.GameInstance
	for _, instance := range allInstances {
		if instance.GameID == gameID {
			instances = append(instances, instance)
		}
	}

	return game, instances, nil
}

// RefreshGames rescans all sources and updates the database
func (s *GamesService) RefreshGames() error {
	s.logger.Info("refreshing games from all sources")

	for _, source := range s.registry.GetAll() {
		s.logger.Info("refreshing source", "source", source.Name())

		instances, err := source.GetInstances(context.Background())
		if err != nil {
			s.logger.Error("failed to get instances from source", "source", source.Name(), "error", err)
			continue
		}

		for _, instance := range instances {
			// Check if instance already exists
			existing, err := s.db.GetInstance(instance.ID)
			if err != nil {
				s.logger.Error("failed to check existing instance", "error", err)
				continue
			}

			if existing == nil {
				// Check if game exists
				game, err := s.db.GetGame(instance.GameID)
				if err != nil {
					s.logger.Error("failed to check game", "error", err)
					continue
				}

				// Create game if not exists
				if game == nil {
					game = &models.Game{
						ID:        instance.GameID,
						Name:      s.getDisplayName(instance),
						Platforms: []string{instance.Platform},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					if err := s.db.CreateGame(game); err != nil {
						s.logger.Error("failed to create game", "error", err)
						continue
					}
				}

				// Create instance
				if err := s.db.CreateInstance(&instance); err != nil {
					s.logger.Error("failed to create instance", "error", err)
					continue
				}

				// Queue metadata fetch
				s.queueMetadataFetch(instance)

				s.logger.Debug("added new instance", "id", instance.ID, "name", game.Name)
			} else {
				// Update existing instance
				updated := false

				// Sync CustomMetadata
				if len(instance.CustomMetadata) > 0 {
					// Check if metadata differs
					needsUpdate := false
					if existing.CustomMetadata == nil {
						needsUpdate = true
					} else {
						for key, value := range instance.CustomMetadata {
							if existing.CustomMetadata[key] != value {
								needsUpdate = true
								break
							}
						}
					}

					if needsUpdate {
						// Merge new metadata with existing
						mergedMetadata := make(map[string]any)
						for k, v := range existing.CustomMetadata {
							mergedMetadata[k] = v
						}
						for k, v := range instance.CustomMetadata {
							mergedMetadata[k] = v
						}

						if err := s.db.UpdateInstanceCustomMetadata(instance.ID, mergedMetadata); err != nil {
							s.logger.Error("failed to update custom metadata", "error", err, "instanceID", instance.ID)
						} else {
							s.logger.Debug("updated custom metadata", "instanceID", instance.ID)
							updated = true
						}
					}
				}

				// Update other instance fields if changed
				if existing.InstallPath != instance.InstallPath ||
					existing.FileSize != instance.FileSize ||
					existing.Installed != instance.Installed {
					existing.InstallPath = instance.InstallPath
					existing.FileSize = instance.FileSize
					existing.Installed = instance.Installed

					if err := s.db.UpdateInstance(existing); err != nil {
						s.logger.Error("failed to update instance", "error", err, "instanceID", instance.ID)
					} else {
						s.logger.Debug("updated instance fields", "instanceID", instance.ID)
						updated = true
					}
				}

				if updated {
					s.logger.Info("synced instance changes", "instanceID", instance.ID, "source", source.Name())
				}
			}
		}
	}

	s.logger.Info("game refresh complete")
	return nil
}

// RefreshSource rescans a specific source
func (s *GamesService) RefreshSource(sourceName string) error {
	source, ok := s.registry.Get(sourceName)
	if !ok {
		return fmt.Errorf("source not found: %s", sourceName)
	}

	if err := source.Refresh(context.Background()); err != nil {
		return fmt.Errorf("failed to refresh source: %w", err)
	}

	return s.RefreshGames()
}

// GetSources returns list of available sources
func (s *GamesService) GetSources() []string {
	return s.registry.GetNames()
}

// UpdateInstanceMetadata updates custom metadata for an instance
func (s *GamesService) UpdateInstanceMetadata(instanceID string, updates map[string]any) error {
	// Cancel any active fetch
	s.fetcher.Cancel(instanceID)

	// Get current metadata
	instance, err := s.db.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}
	if instance == nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	// Merge with existing custom metadata
	if instance.CustomMetadata == nil {
		instance.CustomMetadata = make(map[string]any)
	}
	for key, value := range updates {
		instance.CustomMetadata[key] = value
	}

	// Update in database
	if err := s.db.UpdateInstanceCustomMetadata(instanceID, instance.CustomMetadata); err != nil {
		return fmt.Errorf("failed to update custom metadata: %w", err)
	}

	// Emit update event
	s.emitMetadataUpdate(instanceID, instance.GameID, models.MetadataStatus{
		State:   models.MetadataStateCompleted,
		Message: "User edited",
	})

	return nil
}

// CancelMetadataFetch cancels an active metadata fetch
func (s *GamesService) CancelMetadataFetch(instanceID string) error {
	s.fetcher.Cancel(instanceID)
	return nil
}

// AddManualROM adds a ROM file manually
func (s *GamesService) AddManualROM(filePath string, platform string) (*models.GameInstance, error) {
	// Get file source
	source, ok := s.registry.Get("file")
	if !ok {
		return nil, fmt.Errorf("file source not available")
	}

	// Cast to emulated source to access AddManualROM
	emulatedSource, ok := source.(*emulated.Source)
	if !ok {
		return nil, fmt.Errorf("invalid emulated source type")
	}

	instance, err := emulatedSource.AddManualROM(filePath, platform)
	if err != nil {
		return nil, err
	}

	// Save to database
	game, err := s.db.GetGame(instance.GameID)
	if err != nil {
		return nil, fmt.Errorf("failed to check game: %w", err)
	}
	if game == nil {
		game = &models.Game{
			ID:        instance.GameID,
			Name:      s.getDisplayName(*instance),
			Platforms: []string{instance.Platform},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.db.CreateGame(game); err != nil {
			return nil, fmt.Errorf("failed to create game: %w", err)
		}
	}

	if err := s.db.CreateInstance(instance); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Queue metadata fetch
	s.queueMetadataFetch(*instance)

	return instance, nil
}

// GetArtURL returns the HTTP URL for game art
func (s *GamesService) GetArtURL(instanceID string, artType string) (string, error) {
	if s.route == "" {
		return "", fmt.Errorf("service route not configured")
	}
	return fmt.Sprintf("%s/art/%s/%s", s.route, instanceID, artType), nil
}

// ServeHTTP implements http.Handler for serving game art
func (s *GamesService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /art/{instanceID}/{artType}
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	instanceID := parts[1]
	artType := parts[2]

	// Get instance to find source
	instance, err := s.db.GetInstance(instanceID)
	if err != nil {
		http.Error(w, "Failed to get instance", http.StatusInternalServerError)
		return
	}
	if instance == nil {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	// Get source
	source, ok := s.registry.Get(instance.Source)
	if !ok {
		http.Error(w, "Source not found", http.StatusInternalServerError)
		return
	}

	// Get art from source
	data, contentType, err := source.GetGameArt(r.Context(), instanceID, artType)
	if err != nil {
		// If art not found, try to serve placeholder or 404
		http.Error(w, "Art not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.Write(data)
}

// Helper functions

func (s *GamesService) getDisplayName(instance models.GameInstance) string {
	// Try custom metadata first
	if name, ok := instance.CustomMetadata["name"].(string); ok && name != "" {
		return name
	}

	// Try source data
	if instance.SourceData != nil {
		if name, ok := instance.SourceData["displayName"].(string); ok && name != "" {
			return name
		}
	}

	// Fallback to filename
	return instance.Filename
}

func (s *GamesService) queueMetadataFetch(instance models.GameInstance) {
	req := models.FetchRequest{
		GameID:     instance.GameID,
		InstanceID: instance.ID,
		Priority:   1,
		Platforms:  []string{instance.Platform},
		Name:       s.getDisplayName(instance),
		FileHash:   instance.FileHash,
	}

	// Update status
	s.db.UpdateInstanceMetadataStatus(instance.ID, models.MetadataStatus{
		State:     models.MetadataStateFetching,
		Message:   "Queued for metadata fetching",
		StartedAt: func() *time.Time { t := time.Now(); return &t }(),
	})

	// Emit status update
	s.emitMetadataUpdate(instance.ID, instance.GameID, models.MetadataStatus{
		State:     models.MetadataStateFetching,
		Message:   "Queued for metadata fetching",
		StartedAt: func() *time.Time { t := time.Now(); return &t }(),
	})

	// Queue the request
	if err := s.fetcher.Queue(req); err != nil {
		s.logger.Error("failed to queue metadata fetch", "error", err)
	}
}

func (s *GamesService) emitMetadataUpdate(instanceID, gameID string, status models.MetadataStatus) {
	app := application.Get()
	if app != nil {
		update := models.MetadataStatusUpdate{
			InstanceID: instanceID,
			GameID:     gameID,
			Status:     status,
		}
		app.Event.Emit("metadataStatusUpdate", update)
	}
}

// Launch starts a game instance and monitors its process
func (s *GamesService) Launch(instanceID string) error {
	s.logger.Info("Launch called", "instanceID", instanceID)

	// Lookup instance
	instance, err := s.db.GetInstance(instanceID)
	if err != nil {
		s.logger.Error("failed to get instance", "error", err)
		return fmt.Errorf("failed to get instance: %w", err)
	}
	if instance == nil {
		s.logger.Error("instance not found", "instanceID", instanceID)
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	s.logger.Info("found instance", "instanceID", instance.ID, "gameID", instance.GameID, "source", instance.Source)

	// Emit launching event immediately
	s.logger.Info("emitting launching event")
	s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusLaunching, "")

	// Get source
	source, ok := s.registry.Get(instance.Source)
	if !ok {
		s.logger.Error("unknown source", "source", instance.Source)
		s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusFailed, "unknown source: "+instance.Source)
		return fmt.Errorf("unknown source: %s", instance.Source)
	}

	s.logger.Info("starting async launch", "source", source.Name())

	// Launch async
	go func() {
		ctx := context.Background()

		// Call source launch
		s.logger.Info("calling source.Launch")
		cmd, err := source.Launch(ctx, *instance)
		if err != nil {
			s.logger.Error("source.Launch failed", "error", err)
			s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusFailed, err.Error())
			return
		}

		s.logger.Info("source.Launch succeeded, starting process monitoring")

		// Emit "running" status immediately for emulated games
		// (Steam games emit "running" via activity-based detection in monitorGameProcess)
		if instance.Source == "emulated" {
			s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusRunning, "")
		}

		// Source-specific process monitoring
		// - Emulated: Uses Wait() for immediate exit detection
		// - Steam: Uses activity-based polling (falls back to monitorGameProcess)
		source.MonitorProcess(ctx, *instance, cmd)

		// For sources that need activity-based monitoring (like Steam), also start that
		if instance.Source == "steam" {
			s.monitorGameProcess(instance)
		}
	}()

	return nil
}

// monitorGameProcess monitors the game directory for running executables
func (s *GamesService) monitorGameProcess(instance *models.GameInstance) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	const stopThreshold = 10 * time.Second
	var lastSeenRunning time.Time
	hasBeenRunning := false

	for {
		select {
		case <-ticker.C:
			running, err := s.isProcessRunningInPath(instance.InstallPath)
			if err != nil {
				s.logger.Error("failed to check process status", "error", err)
				continue
			}

			if running {
				// Emit running on first detection
				if !hasBeenRunning {
					s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusRunning, "")
					hasBeenRunning = true
				}
				lastSeenRunning = time.Now()
			} else if hasBeenRunning && time.Since(lastSeenRunning) > stopThreshold {
				// Emit stopped after threshold
				s.emitLaunchStatus(instance.ID, instance.GameID, models.LaunchStatusStopped, "")
				return
			}
		}
	}
}

// normalizeWinePath converts Wine/Proton paths to Linux format
// Handles paths like "Z:\home\user\..." -> "/home/user/..."
func normalizeWinePath(path string) string {
	// Handle Wine/Proton paths with drive letter (e.g., "Z:\home\user\...")
	if len(path) > 2 && path[1] == ':' {
		// Remove drive letter and colon (e.g., "Z:")
		path = path[2:]
	}
	// Convert Windows backslashes to Unix forward slashes
	return strings.ReplaceAll(path, `\`, `/`)
}

// isProcessRunningInPath checks if any process executable is within the install path
func (s *GamesService) isProcessRunningInPath(installPath string) (bool, error) {
	processes, err := process.Processes()
	if err != nil {
		return false, err
	}
	for _, p := range processes {
		// Check exe first (native Linux format)
		exe, err := p.Exe()
		if err == nil && strings.HasPrefix(exe, installPath) {
			return true, nil
		}
		// Check cmdline for Wine/Proton paths
		cmdline, err := p.Cmdline()
		if err == nil {
			normalizedCmdline := normalizeWinePath(cmdline)
			if strings.Contains(normalizedCmdline, installPath) {
				return true, nil
			}
		}
	}
	return false, nil
}

// emitLaunchStatus emits a launch status update event
func (s *GamesService) emitLaunchStatus(instanceID, gameID string, status models.LaunchStatus, errMsg string) {
	app := application.Get()
	if app == nil {
		s.logger.Error("cannot emit launch status: app not available", "instanceID", instanceID, "status", status)
		return
	}

	update := models.LaunchStatusUpdate{
		InstanceID: instanceID,
		GameID:     gameID,
		Status:     status,
		Error:      errMsg,
	}

	s.logger.Info("emitting launch status update", "instanceID", instanceID, "gameID", gameID, "status", status)
	app.Event.Emit("launchStatusUpdate", update)
}

// Emulator API methods for Wails bindings

// GetEmulators returns all configured emulators
func (s *GamesService) GetEmulators() ([]models.Emulator, error) {
	return s.emuService.GetEmulators()
}

// GetEmulatorsForPlatform returns emulators available for a platform
func (s *GamesService) GetEmulatorsForPlatform(platform string) ([]models.Emulator, []models.EmulatorCore, error) {
	return s.emuService.GetEmulatorsForPlatform(platform)
}

// SetPlatformDefaultEmulator sets the default emulator for a platform
func (s *GamesService) SetPlatformDefaultEmulator(platform, emulatorID, coreID string) error {
	return s.emuService.SetPlatformDefault(platform, emulatorID, coreID)
}

// SetInstanceEmulator sets the emulator for a specific game instance
func (s *GamesService) SetInstanceEmulator(instanceID, emulatorID, coreID string) error {
	return s.emuService.SetInstanceEmulator(instanceID, emulatorID, coreID, "")
}

// RefreshEmulators re-discovers available emulators
func (s *GamesService) RefreshEmulators() error {
	return s.emuService.DiscoverAvailable()
}
