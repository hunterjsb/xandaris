package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}
		collection.Name = "fleet_orders"
		collection.Type = "base"
		collection.ListRule = types.Pointer("@request.auth.id = user_id")
		collection.ViewRule = types.Pointer("@request.auth.id = user_id")
		collection.CreateRule = types.Pointer("@request.auth.id != \"\"")
		collection.UpdateRule = types.Pointer(`@request.auth.id = user_id && @request.data.status = "cancelled" && status = "pending"`)
		collection.DeleteRule = types.Pointer(`@request.auth.id = user_id && (status = "failed" || status = "cancelled")`)

		// Schema Fields
		maxSelectOne := 1

		// user_id
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "user_id",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "_users",
				CascadeDelete: false, // User deletion shouldn't cascade to delete orders directly; handle cleanup separately if needed
				MinSelect:     &maxSelectOne,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		})

		// fleet_id
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "fleet_id",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "fleets",
				CascadeDelete: true, // If a fleet is deleted, its orders should be too
				MinSelect:     &maxSelectOne,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		})
		
		// type
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "type",
			Type:     schema.FieldTypeSelect,
			Required: true,
			Options: &schema.SelectOptions{
				MaxSelect: 1,
				Values:    []string{"move"},
			},
		})

		// status
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "status",
			Type:     schema.FieldTypeSelect,
			Required: true,
			Options: &schema.SelectOptions{
				MaxSelect: 1,
				Values:    []string{"pending", "processing", "completed", "failed", "cancelled"},
			},
		})

		// execute_at_tick
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "execute_at_tick",
			Type:     schema.FieldTypeNumber,
			Required: true,
			Options:  &schema.NumberOptions{Min: types.Pointer(0.0)},
		})
		
		// destination_system_id - where the fleet is going
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "destination_system_id",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "systems",
				CascadeDelete: false,
				MinSelect:     &maxSelectOne,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		})

		// original_system_id - where the fleet started from
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "original_system_id",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "systems",
				CascadeDelete: false,
				MinSelect:     &maxSelectOne,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		})

		// travel_time_ticks - how many ticks the journey takes
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "travel_time_ticks",
			Type:     schema.FieldTypeNumber,
			Required: true,
			Options:  &schema.NumberOptions{Min: types.Pointer(1.0)},
		})

		// Indexes
		collection.Indexes = append(collection.Indexes, "CREATE INDEX idx_fleet_orders_user_status ON fleet_orders (user_id, status)")
		collection.Indexes = append(collection.Indexes, "CREATE INDEX idx_fleet_orders_status_execute_at_tick ON fleet_orders (status, execute_at_tick)")
		collection.Indexes = append(collection.Indexes, "CREATE INDEX idx_fleet_orders_fleet_status ON fleet_orders (fleet_id, status)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("fleet_orders")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
