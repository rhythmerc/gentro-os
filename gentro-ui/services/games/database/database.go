package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// DB wraps the SQLite database
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := ensureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates the database schema
func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS games (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			release_date DATETIME,
			developer TEXT,
			publisher TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS game_genres (
			game_id TEXT NOT NULL,
			genre TEXT NOT NULL,
			PRIMARY KEY (game_id, genre),
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS game_platforms (
			game_id TEXT NOT NULL,
			platform TEXT NOT NULL,
			PRIMARY KEY (game_id, platform),
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS game_art (
			game_id TEXT NOT NULL,
			art_type TEXT NOT NULL,
			url TEXT NOT NULL,
			source TEXT NOT NULL,
			PRIMARY KEY (game_id, art_type),
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS game_instances (
			id TEXT PRIMARY KEY,
			game_id TEXT NOT NULL,
			source TEXT NOT NULL,
			platform TEXT NOT NULL,
			source_id TEXT,
			path TEXT,
			filename TEXT,
			file_size INTEGER,
			file_hash TEXT,
			installed BOOLEAN DEFAULT 0,
			install_path TEXT,
			metadata_state TEXT DEFAULT 'idle',
			metadata_message TEXT,
			metadata_error TEXT,
			metadata_started_at DATETIME,
			metadata_completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS instance_custom_metadata (
			instance_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT,
			PRIMARY KEY (instance_id, key),
			FOREIGN KEY (instance_id) REFERENCES game_instances(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS external_metadata (
			game_id TEXT NOT NULL,
			source TEXT NOT NULL,
			data TEXT NOT NULL,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (game_id, source),
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_instances_game_id ON game_instances(game_id)`,
		`CREATE INDEX IF NOT EXISTS idx_instances_source ON game_instances(source)`,
		`CREATE INDEX IF NOT EXISTS idx_instances_platform ON game_instances(platform)`,
		`CREATE INDEX IF NOT EXISTS idx_instances_installed ON game_instances(installed)`,
		// Migration 6: Create emulators table
		`CREATE TABLE IF NOT EXISTS emulators (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL,
			type TEXT NOT NULL,
			executable_path TEXT,
			flatpak_id TEXT,
			command_template TEXT NOT NULL,
			default_args TEXT,
			supported_platforms TEXT,
			is_available BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Migration 7: Create emulator cores (for RetroArch Option B)
		`CREATE TABLE IF NOT EXISTS emulator_cores (
			id TEXT PRIMARY KEY,
			emulator_id TEXT NOT NULL,
			core_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			supported_platforms TEXT,
			is_available BOOLEAN DEFAULT 0,
			FOREIGN KEY (emulator_id) REFERENCES emulators(id) ON DELETE CASCADE
		)`,
		// Migration 8: Create platform_emulators table
		`CREATE TABLE IF NOT EXISTS platform_emulators (
			id TEXT PRIMARY KEY,
			platform TEXT NOT NULL,
			emulator_id TEXT NOT NULL,
			core_id TEXT,
			is_default BOOLEAN DEFAULT 0,
			priority INTEGER DEFAULT 0,
			platform_args TEXT,
			FOREIGN KEY (emulator_id) REFERENCES emulators(id) ON DELETE CASCADE,
			UNIQUE(platform, emulator_id, core_id)
		)`,
		// Migration 9: Create instance_emulator_settings
		`CREATE TABLE IF NOT EXISTS instance_emulator_settings (
			instance_id TEXT PRIMARY KEY,
			emulator_id TEXT NOT NULL,
			core_id TEXT,
			custom_args TEXT,
			FOREIGN KEY (instance_id) REFERENCES game_instances(id) ON DELETE CASCADE,
			FOREIGN KEY (emulator_id) REFERENCES emulators(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

// ensureDir creates the directory if it doesn't exist
func ensureDir(path string) error {
	// Implementation depends on OS - stub for now
	return nil
}

// CreateGame creates a new game record
func (db *DB) CreateGame(game *models.Game) error {
	query := `
		INSERT INTO games (id, name, description, release_date, developer, publisher)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query, game.ID, game.Name, game.Description, game.ReleaseDate, game.Developer, game.Publisher)
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// Insert genres
	if len(game.Genres) > 0 {
		for _, genre := range game.Genres {
			_, err := db.conn.Exec("INSERT INTO game_genres (game_id, genre) VALUES (?, ?)", game.ID, genre)
			if err != nil {
				return fmt.Errorf("failed to insert genre: %w", err)
			}
		}
	}

	// Insert platforms
	if len(game.Platforms) > 0 {
		for _, platform := range game.Platforms {
			_, err := db.conn.Exec("INSERT INTO game_platforms (game_id, platform) VALUES (?, ?)", game.ID, platform)
			if err != nil {
				return fmt.Errorf("failed to insert platform: %w", err)
			}
		}
	}

	return nil
}

// GetGame retrieves a game by ID
func (db *DB) GetGame(id string) (*models.Game, error) {
	game := &models.Game{}
	query := `SELECT id, name, description, release_date, developer, publisher, created_at, updated_at FROM games WHERE id = ?`
	err := db.conn.QueryRow(query, id).Scan(&game.ID, &game.Name, &game.Description, &game.ReleaseDate, &game.Developer, &game.Publisher, &game.CreatedAt, &game.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Load genres
	genres, err := db.getGameGenres(id)
	if err != nil {
		return nil, err
	}
	game.Genres = genres

	// Load platforms
	platforms, err := db.getGamePlatforms(id)
	if err != nil {
		return nil, err
	}
	game.Platforms = platforms

	// Load art URLs
	artURLs, err := db.getGameArt(id)
	if err != nil {
		return nil, err
	}
	game.ArtURLs = artURLs

	return game, nil
}

func (db *DB) getGameGenres(gameID string) ([]string, error) {
	rows, err := db.conn.Query("SELECT genre FROM game_genres WHERE game_id = ?", gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get genres: %w", err)
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var genre string
		if err := rows.Scan(&genre); err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}
	return genres, nil
}

func (db *DB) getGamePlatforms(gameID string) ([]string, error) {
	rows, err := db.conn.Query("SELECT platform FROM game_platforms WHERE game_id = ?", gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get platforms: %w", err)
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var platform string
		if err := rows.Scan(&platform); err != nil {
			return nil, err
		}
		platforms = append(platforms, platform)
	}
	return platforms, nil
}

func (db *DB) getGameArt(gameID string) (map[string]string, error) {
	rows, err := db.conn.Query("SELECT art_type, url FROM game_art WHERE game_id = ?", gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get art: %w", err)
	}
	defer rows.Close()

	artURLs := make(map[string]string)
	for rows.Next() {
		var artType, url string
		if err := rows.Scan(&artType, &url); err != nil {
			return nil, err
		}
		artURLs[artType] = url
	}
	return artURLs, nil
}

// StoreGameArt stores art URL with source for a game
func (db *DB) StoreGameArt(gameID, artType, url, source string) error {
	query := `
		INSERT INTO game_art (game_id, art_type, url, source)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(game_id, art_type) DO UPDATE SET
			url = excluded.url,
			source = excluded.source
	`
	_, err := db.conn.Exec(query, gameID, artType, url, source)
	if err != nil {
		return fmt.Errorf("failed to store game art: %w", err)
	}
	return nil
}

// GetGameArtSource retrieves the source for a specific art type of a game
func (db *DB) GetGameArtSource(gameID, artType string) (string, error) {
	var source string
	err := db.conn.QueryRow("SELECT source FROM game_art WHERE game_id = ? AND art_type = ?", gameID, artType).Scan(&source)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get art source: %w", err)
	}
	return source, nil
}

// CreateInstance creates a new game instance with custom metadata
func (db *DB) CreateInstance(instance *models.GameInstance) error {
	// Start a transaction to ensure atomicity
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the instance
	query := `
		INSERT INTO game_instances (
			id, game_id, source, platform, source_id, path, filename, 
			file_size, file_hash, installed, install_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(query,
		instance.ID, instance.GameID, instance.Source, instance.Platform,
		instance.SourceID, instance.Path, instance.Filename,
		instance.FileSize, instance.FileHash, instance.Installed,
		instance.InstallPath,
	)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	// Insert custom metadata if present
	if len(instance.CustomMetadata) > 0 {
		for key, value := range instance.CustomMetadata {
			valueJSON, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal custom metadata value: %w", err)
			}
			_, err = tx.Exec(
				"INSERT INTO instance_custom_metadata (instance_id, key, value) VALUES (?, ?, ?)",
				instance.ID, key, string(valueJSON),
			)
			if err != nil {
				return fmt.Errorf("failed to insert custom metadata: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetInstance retrieves an instance by ID
func (db *DB) GetInstance(id string) (*models.GameInstance, error) {
	instance := &models.GameInstance{}
	query := `
		SELECT id, game_id, source, platform, source_id, path, filename,
			file_size, file_hash, installed, install_path,
			metadata_state, metadata_message, metadata_error,
			metadata_started_at, metadata_completed_at,
			created_at, updated_at
		FROM game_instances WHERE id = ?
	`
	var metadataState string
	err := db.conn.QueryRow(query, id).Scan(
		&instance.ID, &instance.GameID, &instance.Source, &instance.Platform,
		&instance.SourceID, &instance.Path, &instance.Filename,
		&instance.FileSize, &instance.FileHash, &instance.Installed,
		&instance.InstallPath,
		&metadataState, &instance.MetadataStatus.Message, &instance.MetadataStatus.Error,
		&instance.MetadataStatus.StartedAt, &instance.MetadataStatus.CompletedAt,
		&instance.CreatedAt, &instance.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	instance.MetadataStatus.State = models.MetadataState(metadataState)

	// Load custom metadata
	customMeta, err := db.GetInstanceCustomMetadata(id)
	if err != nil {
		return nil, err
	}
	instance.CustomMetadata = customMeta

	return instance, nil
}

// GetInstanceCustomMetadata retrieves custom metadata for an instance
func (db *DB) GetInstanceCustomMetadata(instanceID string) (map[string]any, error) {
	rows, err := db.conn.Query("SELECT key, value FROM instance_custom_metadata WHERE instance_id = ?", instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom metadata: %w", err)
	}
	defer rows.Close()

	customMeta := make(map[string]any)
	for rows.Next() {
		var key, valueStr string
		if err := rows.Scan(&key, &valueStr); err != nil {
			return nil, err
		}
		var value any
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			// Store as string if not valid JSON
			customMeta[key] = valueStr
		} else {
			customMeta[key] = value
		}
	}
	return customMeta, nil
}

// UpdateInstanceMetadataStatus updates the metadata status
func (db *DB) UpdateInstanceMetadataStatus(instanceID string, status models.MetadataStatus) error {
	query := `
		UPDATE game_instances SET
			metadata_state = ?,
			metadata_message = ?,
			metadata_error = ?,
			metadata_started_at = ?,
			metadata_completed_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query,
		status.State, status.Message, status.Error,
		status.StartedAt, status.CompletedAt, instanceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update metadata status: %w", err)
	}
	return nil
}

// GetInstances retrieves instances matching a filter
func (db *DB) GetInstances(filter models.GameFilter) ([]models.GameInstance, error) {
	// Use LEFT JOIN to load custom metadata in single query
	query := `
		SELECT gi.id, gi.game_id, gi.source, gi.platform, gi.source_id, 
			gi.path, gi.filename, gi.file_size, gi.file_hash, 
			gi.installed, gi.install_path,
			gi.metadata_state, gi.metadata_message, gi.metadata_error,
			gi.metadata_started_at, gi.metadata_completed_at,
			gi.created_at, gi.updated_at,
			icm.key, icm.value
		FROM game_instances gi
		LEFT JOIN instance_custom_metadata icm ON gi.id = icm.instance_id
		WHERE 1=1
	`
	var args []interface{}

	if filter.InstalledOnly {
		query += " AND gi.installed = 1"
	}
	if filter.Source != "" {
		query += " AND gi.source = ?"
		args = append(args, filter.Source)
	}
	if filter.Platform != "" {
		query += " AND gi.platform = ?"
		args = append(args, filter.Platform)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}
	defer rows.Close()

	// Use map to deduplicate instances and accumulate metadata
	instanceMap := make(map[string]*models.GameInstance)

	for rows.Next() {
		instance := models.GameInstance{}
		var metadataState string
		var metaKey, metaValue sql.NullString

		err := rows.Scan(
			&instance.ID, &instance.GameID, &instance.Source, &instance.Platform,
			&instance.SourceID, &instance.Path, &instance.Filename,
			&instance.FileSize, &instance.FileHash, &instance.Installed,
			&instance.InstallPath,
			&metadataState, &instance.MetadataStatus.Message, &instance.MetadataStatus.Error,
			&instance.MetadataStatus.StartedAt, &instance.MetadataStatus.CompletedAt,
			&instance.CreatedAt, &instance.UpdatedAt,
			&metaKey, &metaValue,
		)
		if err != nil {
			return nil, err
		}
		instance.MetadataStatus.State = models.MetadataState(metadataState)

		// Check if we already have this instance
		existing, found := instanceMap[instance.ID]
		if !found {
			// New instance
			instance.CustomMetadata = make(map[string]any)
			instanceMap[instance.ID] = &instance
			existing = &instance
		}

		// Add custom metadata if present
		if metaKey.Valid && metaValue.Valid {
			var value any
			if err := json.Unmarshal([]byte(metaValue.String), &value); err != nil {
				// If unmarshal fails, store as string
				existing.CustomMetadata[metaKey.String] = metaValue.String
			} else {
				existing.CustomMetadata[metaKey.String] = value
			}
		}
	}

	// Convert map to slice
	var instances []models.GameInstance
	for _, instance := range instanceMap {
		instances = append(instances, *instance)
	}

	return instances, nil
}

// FindGameByNameAndPlatform finds a game by name and platform
func (db *DB) FindGameByNameAndPlatform(name string, platform string) (*models.Game, error) {
	var gameID string
	query := `
		SELECT g.id FROM games g
		JOIN game_platforms gp ON g.id = gp.game_id
		WHERE g.name = ? AND gp.platform = ?
		LIMIT 1
	`
	err := db.conn.QueryRow(query, name, platform).Scan(&gameID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find game: %w", err)
	}

	return db.GetGame(gameID)
}

// FindInstanceByPath finds an instance by file path
func (db *DB) FindInstanceByPath(path string) (*models.GameInstance, error) {
	var id string
	err := db.conn.QueryRow("SELECT id FROM game_instances WHERE path = ?", path).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find instance by path: %w", err)
	}
	return db.GetInstance(id)
}

// UpdateGame updates a game record
func (db *DB) UpdateGame(game *models.Game) error {
	query := `
		UPDATE games SET
			name = ?, description = ?, release_date = ?,
			developer = ?, publisher = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, game.Name, game.Description, game.ReleaseDate,
		game.Developer, game.Publisher, game.ID)
	if err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}
	return nil
}

// UpdateInstance updates basic instance fields that may change
func (db *DB) UpdateInstance(instance *models.GameInstance) error {
	query := `
		UPDATE game_instances SET
			path = ?,
			file_size = ?,
			installed = ?,
			install_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query,
		instance.Path,
		instance.FileSize,
		instance.Installed,
		instance.InstallPath,
		instance.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}
	return nil
}

// UpdateInstanceCustomMetadata updates custom metadata for an instance
func (db *DB) UpdateInstanceCustomMetadata(instanceID string, metadata map[string]any) error {
	// Delete existing custom metadata
	_, err := db.conn.Exec("DELETE FROM instance_custom_metadata WHERE instance_id = ?", instanceID)
	if err != nil {
		return fmt.Errorf("failed to clear custom metadata: %w", err)
	}

	// Insert new values
	for key, value := range metadata {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata value: %w", err)
		}
		_, err = db.conn.Exec(
			"INSERT INTO instance_custom_metadata (instance_id, key, value) VALUES (?, ?, ?)",
			instanceID, key, string(valueJSON),
		)
		if err != nil {
			return fmt.Errorf("failed to insert custom metadata: %w", err)
		}
	}

	return nil
}

// StoreExternalMetadata stores metadata from an external source
func (db *DB) StoreExternalMetadata(gameID string, source string, data map[string]any) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal external metadata: %w", err)
	}

	query := `
		INSERT INTO external_metadata (game_id, source, data)
		VALUES (?, ?, ?)
		ON CONFLICT(game_id, source) DO UPDATE SET
			data = excluded.data,
			fetched_at = CURRENT_TIMESTAMP
	`
	_, err = db.conn.Exec(query, gameID, source, string(dataJSON))
	if err != nil {
		return fmt.Errorf("failed to store external metadata: %w", err)
	}
	return nil
}

// GetExternalMetadata retrieves cached metadata from an external source
func (db *DB) GetExternalMetadata(gameID string, source string) (map[string]any, error) {
	query := `
		SELECT data FROM external_metadata
		WHERE game_id = ? AND source = ?
	`
	var dataJSON string
	err := db.conn.QueryRow(query, gameID, source).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get external metadata: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal external metadata: %w", err)
	}

	return data, nil
}

// Emulator methods

// UpsertEmulator creates or updates an emulator record
func (db *DB) UpsertEmulator(emu models.Emulator) error {
	platformsJSON, _ := json.Marshal(emu.SupportedPlatforms)
	query := `
		INSERT INTO emulators (id, name, display_name, type, executable_path, flatpak_id, command_template, default_args, supported_platforms, is_available)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			display_name = excluded.display_name,
			type = excluded.type,
			executable_path = excluded.executable_path,
			flatpak_id = excluded.flatpak_id,
			command_template = excluded.command_template,
			default_args = excluded.default_args,
			supported_platforms = excluded.supported_platforms
	`
	_, err := db.conn.Exec(query, emu.ID, emu.Name, emu.DisplayName, emu.Type, emu.ExecutablePath, emu.FlatpakID, emu.CommandTemplate, emu.DefaultArgs, string(platformsJSON), emu.IsAvailable)
	return err
}

// GetEmulator retrieves an emulator by ID
func (db *DB) GetEmulator(id string) (*models.Emulator, error) {
	query := `SELECT id, name, display_name, type, executable_path, flatpak_id, command_template, default_args, supported_platforms, is_available, created_at, updated_at FROM emulators WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var emu models.Emulator
	var platformsJSON string
	err := row.Scan(&emu.ID, &emu.Name, &emu.DisplayName, &emu.Type, &emu.ExecutablePath, &emu.FlatpakID, &emu.CommandTemplate, &emu.DefaultArgs, &platformsJSON, &emu.IsAvailable, &emu.CreatedAt, &emu.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if platformsJSON != "" {
		json.Unmarshal([]byte(platformsJSON), &emu.SupportedPlatforms)
	}
	return &emu, nil
}

// GetEmulators retrieves all emulators
func (db *DB) GetEmulators() ([]models.Emulator, error) {
	query := `SELECT id, name, display_name, type, executable_path, flatpak_id, command_template, default_args, supported_platforms, is_available, created_at, updated_at FROM emulators`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emulators []models.Emulator
	for rows.Next() {
		var emu models.Emulator
		var platformsJSON string
		err := rows.Scan(&emu.ID, &emu.Name, &emu.DisplayName, &emu.Type, &emu.ExecutablePath, &emu.FlatpakID, &emu.CommandTemplate, &emu.DefaultArgs, &platformsJSON, &emu.IsAvailable, &emu.CreatedAt, &emu.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if platformsJSON != "" {
			json.Unmarshal([]byte(platformsJSON), &emu.SupportedPlatforms)
		}
		emulators = append(emulators, emu)
	}
	return emulators, nil
}

// UpdateEmulatorAvailability updates the availability status of an emulator
func (db *DB) UpdateEmulatorAvailability(id string, available bool) error {
	query := `UPDATE emulators SET is_available = ? WHERE id = ?`
	_, err := db.conn.Exec(query, available, id)
	return err
}

// EmulatorCore methods

// UpsertEmulatorCore creates or updates an emulator core record
func (db *DB) UpsertEmulatorCore(core models.EmulatorCore) error {
	platformsJSON, _ := json.Marshal(core.SupportedPlatforms)
	query := `
		INSERT INTO emulator_cores (id, emulator_id, core_id, display_name, supported_platforms, is_available)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			emulator_id = excluded.emulator_id,
			core_id = excluded.core_id,
			display_name = excluded.display_name,
			supported_platforms = excluded.supported_platforms
	`
	_, err := db.conn.Exec(query, core.ID, core.EmulatorID, core.CoreID, core.DisplayName, string(platformsJSON), core.IsAvailable)
	return err
}

// GetEmulatorCores retrieves all cores for an emulator, or all cores if emulatorID is empty
func (db *DB) GetEmulatorCores(emulatorID string) ([]models.EmulatorCore, error) {
	var query string
	var rows *sql.Rows
	var err error

	if emulatorID == "" {
		query = `SELECT id, emulator_id, core_id, display_name, supported_platforms, is_available FROM emulator_cores`
		rows, err = db.conn.Query(query)
	} else {
		query = `SELECT id, emulator_id, core_id, display_name, supported_platforms, is_available FROM emulator_cores WHERE emulator_id = ?`
		rows, err = db.conn.Query(query, emulatorID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cores []models.EmulatorCore
	for rows.Next() {
		var core models.EmulatorCore
		var platformsJSON string
		err := rows.Scan(&core.ID, &core.EmulatorID, &core.CoreID, &core.DisplayName, &platformsJSON, &core.IsAvailable)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(platformsJSON), &core.SupportedPlatforms)
		cores = append(cores, core)
	}
	return cores, nil
}

// GetCore retrieves a specific core by emulator ID and core ID
func (db *DB) GetCore(emulatorID, coreID string) (*models.EmulatorCore, error) {
	query := `SELECT id, emulator_id, core_id, display_name, supported_platforms, is_available FROM emulator_cores WHERE emulator_id = ? AND core_id = ?`
	row := db.conn.QueryRow(query, emulatorID, coreID)

	var core models.EmulatorCore
	var platformsJSON string
	err := row.Scan(&core.ID, &core.EmulatorID, &core.CoreID, &core.DisplayName, &platformsJSON, &core.IsAvailable)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(platformsJSON), &core.SupportedPlatforms)
	return &core, nil
}

// UpdateEmulatorCoreAvailability updates the availability status of a core
func (db *DB) UpdateEmulatorCoreAvailability(id string, available bool) error {
	query := `UPDATE emulator_cores SET is_available = ? WHERE id = ?`
	_, err := db.conn.Exec(query, available, id)
	return err
}

// PlatformEmulator methods

// UpsertPlatformEmulator creates or updates a platform-emulator mapping
func (db *DB) UpsertPlatformEmulator(pe models.PlatformEmulator) error {
	query := `
		INSERT INTO platform_emulators (id, platform, emulator_id, core_id, is_default)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			platform = excluded.platform,
			emulator_id = excluded.emulator_id,
			core_id = excluded.core_id,
			is_default = excluded.is_default
	`
	_, err := db.conn.Exec(query, pe.ID, pe.Platform, pe.EmulatorID, pe.CoreID, pe.IsDefault)
	return err
}

// ClearPlatformEmulators removes all platform-emulator mappings
func (db *DB) ClearPlatformEmulators() error {
	_, err := db.conn.Exec("DELETE FROM platform_emulators")
	return err
}

// GetDefaultEmulatorForPlatform retrieves the default emulator for a platform
// If requireAvailable is true, only returns emulators marked as available
func (db *DB) GetDefaultEmulatorForPlatform(platform string, requireAvailable bool) (*models.Emulator, *models.EmulatorCore, error) {
	query := `
		SELECT e.id, e.name, e.display_name, e.type, e.executable_path, e.flatpak_id, e.command_template, e.default_args, e.is_available, e.created_at, e.updated_at,
			c.id, c.emulator_id, c.core_id, c.display_name, c.supported_platforms, c.is_available
		FROM emulators e
		JOIN platform_emulators pe ON e.id = pe.emulator_id
		LEFT JOIN emulator_cores c ON pe.core_id = c.core_id AND c.emulator_id = e.id
		WHERE pe.platform = ? AND pe.is_default = 1
	`

	// Add availability filter if required
	if requireAvailable {
		query += ` AND e.is_available = 1 AND (c.core_id IS NULL OR c.is_available = 1)`
	}

	row := db.conn.QueryRow(query, platform)

	var emu models.Emulator
	var core models.EmulatorCore
	var platformsJSON sql.NullString
	var coreDBID sql.NullString
	var coreID sql.NullString
	var emulatorID sql.NullString
	var coreDisplayName sql.NullString
	var coreIsAvailable sql.NullBool

	err := row.Scan(
		&emu.ID, &emu.Name, &emu.DisplayName, &emu.Type, &emu.ExecutablePath, &emu.FlatpakID, &emu.CommandTemplate, &emu.DefaultArgs, &emu.IsAvailable, &emu.CreatedAt, &emu.UpdatedAt,
		&coreDBID, &emulatorID, &coreID, &coreDisplayName, &platformsJSON, &coreIsAvailable,
	)
	if err != nil {
		return nil, nil, err
	}

	if coreID.Valid {
		core.ID = coreDBID.String
		core.EmulatorID = emulatorID.String
		core.CoreID = coreID.String
		core.DisplayName = coreDisplayName.String
		if coreIsAvailable.Valid {
			core.IsAvailable = coreIsAvailable.Bool
		}
		if platformsJSON.Valid {
			json.Unmarshal([]byte(platformsJSON.String), &core.SupportedPlatforms)
		}
		return &emu, &core, nil
	}

	return &emu, nil, nil
}

// GetEmulatorsForPlatform retrieves all emulators available for a platform
func (db *DB) GetEmulatorsForPlatform(platform string) ([]models.Emulator, []models.EmulatorCore, error) {
	query := `
		SELECT e.id, e.name, e.display_name, e.type, e.executable_path, e.flatpak_id, e.command_template, e.default_args, e.is_available, e.created_at, e.updated_at,
			c.id, c.emulator_id, c.core_id, c.display_name, c.supported_platforms, c.is_available
		FROM emulators e
		JOIN platform_emulators pe ON e.id = pe.emulator_id
		LEFT JOIN emulator_cores c ON pe.core_id = c.core_id AND c.emulator_id = e.id
		WHERE pe.platform = ?
		ORDER BY pe.priority ASC
	`
	rows, err := db.conn.Query(query, platform)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var emulators []models.Emulator
	var cores []models.EmulatorCore
	for rows.Next() {
		var emu models.Emulator
		var core models.EmulatorCore
		var platformsJSON sql.NullString
		var coreDBID sql.NullString
		var coreID sql.NullString
		var emulatorID sql.NullString
		var coreDisplayName sql.NullString
		var coreIsAvailable sql.NullBool

		err := rows.Scan(
			&emu.ID, &emu.Name, &emu.DisplayName, &emu.Type, &emu.ExecutablePath, &emu.FlatpakID, &emu.CommandTemplate, &emu.DefaultArgs, &emu.IsAvailable, &emu.CreatedAt, &emu.UpdatedAt,
			&coreDBID, &emulatorID, &coreID, &coreDisplayName, &platformsJSON, &coreIsAvailable,
		)
		if err != nil {
			return nil, nil, err
		}

		emulators = append(emulators, emu)
		if coreID.Valid {
			core.ID = coreDBID.String
			core.EmulatorID = emulatorID.String
			core.CoreID = coreID.String
			core.DisplayName = coreDisplayName.String
			if coreIsAvailable.Valid {
				core.IsAvailable = coreIsAvailable.Bool
			}
			if platformsJSON.Valid {
				json.Unmarshal([]byte(platformsJSON.String), &core.SupportedPlatforms)
			}
			cores = append(cores, core)
		}
	}
	return emulators, cores, nil
}

// AvailableEmulatorPair represents an emulator and its optional core
type AvailableEmulatorPair struct {
	Emulator models.Emulator
	Core     *models.EmulatorCore
}

// GetAvailableEmulatorsForPlatform retrieves all available emulators for a platform
// Returns only emulators marked as available, and for emulators with cores, only if core is also available
func (db *DB) GetAvailableEmulatorsForPlatform(platform string) ([]AvailableEmulatorPair, error) {
	query := `
		SELECT e.id, e.name, e.display_name, e.type, e.executable_path, e.flatpak_id, e.command_template, e.default_args, e.is_available, e.created_at, e.updated_at,
			c.id, c.emulator_id, c.core_id, c.display_name, c.supported_platforms, c.is_available
		FROM emulators e
		JOIN platform_emulators pe ON e.id = pe.emulator_id
		LEFT JOIN emulator_cores c ON pe.core_id = c.core_id AND c.emulator_id = e.id
		WHERE pe.platform = ?
			AND e.is_available = 1
			AND (c.core_id IS NULL OR c.is_available = 1)
		ORDER BY pe.priority ASC
	`
	rows, err := db.conn.Query(query, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get available emulators: %w", err)
	}
	defer rows.Close()

	var pairs []AvailableEmulatorPair
	for rows.Next() {
		var emu models.Emulator
		var core models.EmulatorCore
		var platformsJSON sql.NullString
		var coreDBID sql.NullString
		var coreID sql.NullString
		var emulatorID sql.NullString
		var coreDisplayName sql.NullString
		var coreIsAvailable sql.NullBool

		err := rows.Scan(
			&emu.ID, &emu.Name, &emu.DisplayName, &emu.Type, &emu.ExecutablePath, &emu.FlatpakID, &emu.CommandTemplate, &emu.DefaultArgs, &emu.IsAvailable, &emu.CreatedAt, &emu.UpdatedAt,
			&coreDBID, &emulatorID, &coreID, &coreDisplayName, &platformsJSON, &coreIsAvailable,
		)
		if err != nil {
			return nil, err
		}

		pair := AvailableEmulatorPair{
			Emulator: emu,
			Core:     nil,
		}

		if coreID.Valid {
			core.ID = coreDBID.String
			core.EmulatorID = emulatorID.String
			core.CoreID = coreID.String
			core.DisplayName = coreDisplayName.String
			if coreIsAvailable.Valid {
				core.IsAvailable = coreIsAvailable.Bool
			}
			if platformsJSON.Valid {
				json.Unmarshal([]byte(platformsJSON.String), &core.SupportedPlatforms)
			}
			pair.Core = &core
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// SetPlatformDefaultEmulator sets the default emulator for a platform
func (db *DB) SetPlatformDefaultEmulator(platform, emulatorID, coreID string) error {
	// Clear existing default
	_, err := db.conn.Exec(`UPDATE platform_emulators SET is_default = 0 WHERE platform = ?`, platform)
	if err != nil {
		return err
	}

	// Set new default
	_, err = db.conn.Exec(`UPDATE platform_emulators SET is_default = 1 WHERE platform = ? AND emulator_id = ? AND core_id = ?`, platform, emulatorID, coreID)
	return err
}

// InstanceEmulatorSettings methods

// SetInstanceEmulatorSettings creates or updates instance-specific emulator settings
func (db *DB) SetInstanceEmulatorSettings(instanceID, emulatorID, coreID, customArgs string) error {
	query := `
		INSERT INTO instance_emulator_settings (instance_id, emulator_id, core_id, custom_args)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(instance_id) DO UPDATE SET
			emulator_id = excluded.emulator_id,
			core_id = excluded.core_id,
			custom_args = excluded.custom_args
	`
	_, err := db.conn.Exec(query, instanceID, emulatorID, coreID, customArgs)
	return err
}

// GetInstanceEmulatorSettings retrieves emulator settings for an instance
func (db *DB) GetInstanceEmulatorSettings(instanceID string) (*models.InstanceEmulatorSettings, error) {
	query := `SELECT instance_id, emulator_id, core_id, custom_args FROM instance_emulator_settings WHERE instance_id = ?`
	row := db.conn.QueryRow(query, instanceID)

	var settings models.InstanceEmulatorSettings
	err := row.Scan(&settings.InstanceID, &settings.EmulatorID, &settings.CoreID, &settings.CustomArgs)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}
