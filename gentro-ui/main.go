package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/rhythmerc/gentro-ui/services/games"
	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[models.MetadataStatusUpdate]("metadata:status-update")
	application.RegisterEvent[models.LaunchStatusUpdate]("launchStatusUpdate")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create GamesService
	gamesService, err := games.NewGamesService(games.GamesServiceConfig{
		Logger: logger,
	})
	if err != nil {
		log.Fatalf("Failed to create GamesService: %v", err)
	}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "gentro-ui",
		Description: "A modular game library manager",
		Services: []application.Service{
			application.NewServiceWithOptions(gamesService, application.ServiceOptions{
				Route: "/games",
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Gentro",
		BackgroundColour: application.NewRGB(27, 38, 54),
		StartState:       application.WindowStateFullscreen,
	})

	// Run the application. This blocks until the application has been exited.
	// If an error occurred while running the application, log it and exit.
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}
