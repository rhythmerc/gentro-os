package events

import (
	"log/slog"

	"github.com/rhythmerc/gentro-ui/services/games/models"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Events struct {
	logger *slog.Logger
}

func NewEvents(logger *slog.Logger) *Events {
	return &Events{logger}
}

func (e *Events) EmitGameInstanceRunning(instance models.GameInstance) {
	app := application.Get()
	if app != nil {
		update := models.LaunchStatusUpdate{
			InstanceID: instance.ID,
			GameID:     instance.GameID,
			Status:     models.LaunchStatusRunning,
		}
		app.Event.Emit("launchStatusUpdate", update)
	}

	if e.logger != nil {
		e.logger.Info("game running",
			"instanceId", instance.ID,
			"gameId", instance.GameID,
		)
	}
}

// emitStopped emits a stopped status update
func (e *Events) EmitGameInstanceStopped(instance models.GameInstance) {
	app := application.Get()
	if app != nil {
		update := models.LaunchStatusUpdate{
			InstanceID: instance.ID,
			GameID:     instance.GameID,
			Status:     models.LaunchStatusStopped,
		}
		app.Event.Emit("launchStatusUpdate", update)
	}

	if e.logger != nil {
		e.logger.Info("game stopped",
			"instanceId", instance.ID,
			"gameId", instance.GameID,
		)
	}
}

func (e *Events) EmitGameArtUpdated(instance models.GameInstance) {
}
