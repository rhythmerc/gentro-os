package igdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// Resolver implements the metadata.Resolver interface for IGDB
type Resolver struct {
	client *Client
	logger *slog.Logger
}

// NewResolver creates a new IGDB resolver
func NewResolver(clientID, clientSecret string, logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.Default()
	}

	return &Resolver{
		client: NewClient(clientID, clientSecret),
		logger: logger,
	}
}

// Name returns the resolver name
func (r *Resolver) Name() string {
	return "igdb"
}

// Supports returns true for emulated games on supported platforms
func (r *Resolver) Supports(source, platform string) bool {
	// Only support emulated games (not Steam)
	if source != "emulated" {
		return false
	}

	// Check if platform is supported
	_, supported := PlatformIDs[strings.ToLower(platform)]
	return supported
}

// Resolve fetches metadata from IGDB
func (r *Resolver) Resolve(ctx context.Context, req models.FetchRequest) (models.ResolvedMetadata, error) {
	result := models.ResolvedMetadata{
		GameMetadata:     models.GameMetadata{},
		PlatformMetadata: make(map[string]models.PlatformMetadata),
		ArtURLs:          make(map[string]string),
	}

	// Get platform ID
	platformID, ok := PlatformIDs[strings.ToLower(req.Platform)]
	if !ok {
		return result, fmt.Errorf("unsupported platform: %s", req.Platform)
	}

	r.logger.Info("searching IGDB for game",
		"name", req.Name,
		"platform", req.Platform,
		"platformID", platformID,
	)

	// Search for the game
	game, err := r.client.SearchGame(req.Name, platformID)
	if err != nil {
		return result, fmt.Errorf("failed to search game: %w", err)
	}

	r.logger.Info("found game on IGDB",
		"gameID", game.ID,
		"name", game.Name,
	)

	// Fill in basic metadata
	result.GameMetadata.Name = game.Name
	result.GameMetadata.Description = game.Summary

	if game.ReleaseDate > 0 {
		releaseDate := time.Unix(game.ReleaseDate, 0)
		result.GameMetadata.ReleaseDate = &releaseDate
	}

	// Fetch genres
	if len(game.Genres) > 0 {
		genres, err := r.client.GetGenres(game.Genres)
		if err != nil {
			r.logger.Warn("failed to fetch genres", "error", err)
		} else {
			for _, g := range genres {
				result.GameMetadata.Genres = append(result.GameMetadata.Genres, g.Name)
			}
		}
	}

	// Fetch companies (developers/publishers)
	if len(game.Developers) > 0 {
		// Note: IGDB's involved_companies is more complex, this is simplified
		// In a full implementation, we'd fetch involved_companies first
		r.logger.Info("developers found but not fully implemented", "count", len(game.Developers))
	}

	// Fetch cover art
	if game.Cover > 0 {
		cover, err := r.client.GetCover(game.Cover)
		if err != nil {
			r.logger.Warn("failed to fetch cover", "error", err)
		} else if cover.URL != "" {
			// IGDB URLs need to be converted to full URLs
			result.ArtURLs["cover"] = expandImageURL(cover.URL)
		}
	}

	// Fetch screenshots
	if len(game.Screenshots) > 0 {
		screenshots, err := r.client.GetScreenshots(game.ID)
		if err != nil {
			r.logger.Warn("failed to fetch screenshots", "error", err)
		} else if len(screenshots) > 0 {
			// Use first screenshot as library art
			result.ArtURLs["screenshot"] = expandImageURL(screenshots[0].URL)
		}
	}

	// Fetch artworks (hero images)
	if len(game.Artworks) > 0 {
		artworks, err := r.client.GetArtworks(game.ID)
		if err != nil {
			r.logger.Warn("failed to fetch artworks", "error", err)
		} else if len(artworks) > 0 {
			result.ArtURLs["artwork"] = expandImageURL(artworks[0].URL)
		}
	}

	// Fetch logos
	logos, err := r.client.GetLogos(game.ID)
	if err != nil {
		r.logger.Warn("failed to fetch logos", "error", err)
	} else if len(logos) > 0 {
		// Prioritize: color (7) > white (5) > black (6)
		var colorLogo, whiteLogo, blackLogo *Logo
		for i := range logos {
			switch logos[i].ArtworkType {
			case 7:
				colorLogo = &logos[i]
			case 5:
				whiteLogo = &logos[i]
			case 6:
				blackLogo = &logos[i]
			}
		}

		// Select best available logo
		var selectedLogo *Logo
		if colorLogo != nil {
			selectedLogo = colorLogo
			r.logger.Debug("using color logo", "game", game.Name)
		} else if whiteLogo != nil {
			selectedLogo = whiteLogo
			r.logger.Debug("using white logo", "game", game.Name)
		} else if blackLogo != nil {
			selectedLogo = blackLogo
			r.logger.Debug("using black logo", "game", game.Name)
		}

		if selectedLogo != nil {
			result.ArtURLs["logo"] = expandImageURL(selectedLogo.URL)
		}
	}

	// Set platform-specific metadata
	result.PlatformMetadata[req.Platform] = models.PlatformMetadata{
		Platform: req.Platform,
	}

	r.logger.Info("successfully resolved metadata from IGDB",
		"game", game.Name,
		"genres", len(result.GameMetadata.Genres),
		"art", len(result.ArtURLs),
	)

	return result, nil
}

// expandImageURL converts IGDB's image URL format to a full URL
// and replaces size modifiers with t_720p to get a high resolution image
func expandImageURL(url string) string {
	// IGDB returns URLs like "//images.igdb.com/..."
	// We need to add https: prefix
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	}

	// IGDB URLs contain size modifiers like /t_thumb/, /t_cover_big/, etc.
	// To get the original resolution, replace the size modifier with t_720p
	// Format: https://images.igdb.com/igdb/image/upload/t_size/filename.jpg
	// Full res: https://images.igdb.com/igdb/image/upload/t_720p/filename.jpg
	url = strings.Replace(url, "t_thumb", "t_720p", 1)
	url, isJpeg := strings.CutSuffix(url, ".jpg")
	if isJpeg {
		url = url + ".png"
	}

	return url
}
