package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	core.AppMigrations.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("employee_detections")
		if err != nil {
			return err
		}

		// Add is_target_device field
		collection.Fields.Add(&core.BoolField{
			Id:   "det_target",
			Name: "is_target_device",
		})

		// Add device_name field
		collection.Fields.Add(&core.TextField{
			Id:   "det_name",
			Name: "device_name",
			Max:  255,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("employee_detections")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("det_target")
		collection.Fields.RemoveById("det_name")

		return app.Save(collection)
	})
}
