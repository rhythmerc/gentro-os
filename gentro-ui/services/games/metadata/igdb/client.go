package igdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	twitchAuthURL = "https://id.twitch.tv/oauth2/token"
	igdbBaseURL   = "https://api.igdb.com/v4"
)

// PlatformIDs maps our platform names to IGDB platform IDs
var PlatformIDs = map[string]int{
	"nes":       18,
	"snes":      19,
	"n64":       4,
	"gamecube":  21,
	"wii":       5,
	"ps1":       7,
	"ps2":       8,
	"genesis":   29,
	"saturn":    32,
	"dreamcast": 23,
	"gba":       24,
	"nds":       20,
	"3ds":       37,
	"psp":       38,
}

// Client handles IGDB API communication
type Client struct {
	clientID     string
	clientSecret string
	accessToken  string
	expiresAt    time.Time
	httpClient   *http.Client
}

// Game represents an IGDB game result
type Game struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	ReleaseDate int64  `json:"first_release_date"`
	Developers  []int  `json:"involved_companies"`
	Genres      []int  `json:"genres"`
	Cover       int    `json:"cover"`
	Screenshots []int  `json:"screenshots"`
	Artworks    []int  `json:"artworks"`
}

// Cover represents an IGDB cover image
type Cover struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Game int    `json:"game"`
}

// Screenshot represents an IGDB screenshot
type Screenshot struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Game int    `json:"game"`
}

// Artwork represents an IGDB artwork
type Artwork struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Game        int    `json:"game"`
	ArtworkType int    `json:"artwork_type"`
}

// Logo represents an IGDB logo (filtered artwork with artwork_type)
type Logo struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Game        int    `json:"game"`
	ArtworkType int    `json:"artwork_type"`
}

// Company represents a game company
type Company struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Genre represents a game genre
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// NewClient creates a new IGDB client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// authenticate obtains a Twitch access token
func (c *Client) authenticate() error {
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return nil
	}

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("grant_type", "client_credentials")

	resp, err := c.httpClient.PostForm(twitchAuthURL, data)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Twitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.accessToken = result.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	return nil
}

// SearchGame searches for a game by name and platform
func (c *Client) SearchGame(name string, platformID int) (*Game, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`fields id, name, summary, first_release_date, involved_companies, genres, cover, screenshots, artworks;
		where name ~ "%s" & platforms = (%d);
		limit 1;`,
		escapeQuery(name), platformID,
	)

	games, err := c.queryGames(query)
	if err != nil {
		return nil, err
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no game found for '%s' on platform %d", name, platformID)
	}

	return &games[0], nil
}

// GetGameByID retrieves a game by its IGDB ID
func (c *Client) GetGameByID(gameID int) (*Game, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`fields id, name, summary, first_release_date, involved_companies, genres, cover, screenshots, artworks;
		where id = %d;`,
		gameID,
	)

	games, err := c.queryGames(query)
	if err != nil {
		return nil, err
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("game not found with ID %d", gameID)
	}

	return &games[0], nil
}

// GetCover retrieves cover art for a game
func (c *Client) GetCover(coverID int) (*Cover, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`fields id, url, game;
		where id = %d;`,
		coverID,
	)

	covers, err := c.queryCovers(query)
	if err != nil {
		return nil, err
	}

	if len(covers) == 0 {
		return nil, fmt.Errorf("cover not found with ID %d", coverID)
	}

	return &covers[0], nil
}

// GetScreenshots retrieves screenshots for a game
func (c *Client) GetScreenshots(gameID int) ([]Screenshot, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`fields id, url, game;
		where game = %d;`,
		gameID,
	)

	return c.queryScreenshots(query)
}

// GetArtworks retrieves artworks for a game
func (c *Client) GetArtworks(gameID int) ([]Artwork, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`fields id, url, game, artwork_type;
		where game = %d;`,
		gameID,
	)

	return c.queryArtworks(query)
}

// GetLogos retrieves logos for a game (artworks with logo artwork_type: 5=white, 6=black, 7=color)
func (c *Client) GetLogos(gameID int) ([]Logo, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	// Query for artworks with logo artwork_type IDs (5, 6, 7)
	query := fmt.Sprintf(
		`fields id, url, game, artwork_type;
		where game = %d & artwork_type = (5,6,7);`,
		gameID,
	)

	return c.queryLogos(query)
}

// GetCompanies retrieves company names by IDs
func (c *Client) GetCompanies(companyIDs []int) ([]Company, error) {
	if len(companyIDs) == 0 {
		return nil, nil
	}

	if err := c.authenticate(); err != nil {
		return nil, err
	}

	idsStr := joinInts(companyIDs)
	query := fmt.Sprintf(
		`fields id, name;
		where id = (%s);`,
		idsStr,
	)

	return c.queryCompanies(query)
}

// GetGenres retrieves genre names by IDs
func (c *Client) GetGenres(genreIDs []int) ([]Genre, error) {
	if len(genreIDs) == 0 {
		return nil, nil
	}

	if err := c.authenticate(); err != nil {
		return nil, err
	}

	idsStr := joinInts(genreIDs)
	query := fmt.Sprintf(
		`fields id, name;
		where id = (%s);`,
		idsStr,
	)

	return c.queryGenres(query)
}

// queryGames executes a games query
func (c *Client) queryGames(query string) ([]Game, error) {
	var games []Game
	if err := c.executeQuery("/games", query, &games); err != nil {
		return nil, err
	}
	return games, nil
}

// queryCovers executes a covers query
func (c *Client) queryCovers(query string) ([]Cover, error) {
	var covers []Cover
	if err := c.executeQuery("/covers", query, &covers); err != nil {
		return nil, err
	}
	return covers, nil
}

// queryScreenshots executes a screenshots query
func (c *Client) queryScreenshots(query string) ([]Screenshot, error) {
	var screenshots []Screenshot
	if err := c.executeQuery("/screenshots", query, &screenshots); err != nil {
		return nil, err
	}
	return screenshots, nil
}

// queryArtworks executes an artworks query
func (c *Client) queryArtworks(query string) ([]Artwork, error) {
	var artworks []Artwork
	if err := c.executeQuery("/artworks", query, &artworks); err != nil {
		return nil, err
	}
	return artworks, nil
}

// queryLogos executes a logos query (artworks filtered by type)
func (c *Client) queryLogos(query string) ([]Logo, error) {
	var logos []Logo
	if err := c.executeQuery("/artworks", query, &logos); err != nil {
		return nil, err
	}
	return logos, nil
}

// queryCompanies executes a companies query
func (c *Client) queryCompanies(query string) ([]Company, error) {
	var companies []Company
	if err := c.executeQuery("/companies", query, &companies); err != nil {
		return nil, err
	}
	return companies, nil
}

// queryGenres executes a genres query
func (c *Client) queryGenres(query string) ([]Genre, error) {
	var genres []Genre
	if err := c.executeQuery("/genres", query, &genres); err != nil {
		return nil, err
	}
	return genres, nil
}

// executeQuery executes an IGDB API query
func (c *Client) executeQuery(endpoint, query string, result interface{}) error {
	url := igdbBaseURL + endpoint

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(query))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("query failed: %s (status %d)", string(body), resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// escapeQuery escapes special characters in IGDB queries
func escapeQuery(s string) string {
	// Basic escaping for IGDB query syntax
	// Remove or escape quotes that could break the query
	result := ""
	for _, r := range s {
		switch r {
		case '"', '\\':
			// Skip problematic characters
			continue
		default:
			result += string(r)
		}
	}
	return result
}

// joinInts joins integers into a comma-separated string
func joinInts(nums []int) string {
	result := ""
	for i, n := range nums {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d", n)
	}
	return result
}
