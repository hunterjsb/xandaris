package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("fleets")
		if err != nil {
			return err
		}

		// Remove legacy movement fields - fleet_orders now handles all movement
		collection.Schema.RemoveField("destination_system")
		collection.Schema.RemoveField("eta")
		collection.Schema.RemoveField("next_stop")
		collection.Schema.RemoveField("final_destination")
		collection.Schema.RemoveField("route_path")
		collection.Schema.RemoveField("current_hop")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		// No rollback - moving forward with new fleet orders architecture
		return nil
	})
}