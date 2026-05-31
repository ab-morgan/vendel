package hooks

import (
	"log/slog"
	"vendel/middleware"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterConfigHooks registers hooks for the system_config collection:
// syncing app_name to PocketBase settings and invalidating the maintenance cache.
func RegisterConfigHooks(app *pocketbase.PocketBase) {
	app.OnRecordAfterUpdateSuccess("system_config").BindFunc(func(e *core.RecordEvent) error {
		key := e.Record.GetString("key")

		if key == "app_name" {
			settings := e.App.Settings()
			settings.Meta.AppName = e.Record.GetString("value")
			settings.Meta.SenderName = e.Record.GetString("value")
			if err := e.App.Save(settings); err != nil {
				e.App.Logger().Warn("could not sync app_name to PocketBase settings", slog.Any("error", err))
			}
		}

		if key == "maintenance_mode" {
			middleware.InvalidateMaintenanceCache()
		}

		return e.Next()
	})
}
