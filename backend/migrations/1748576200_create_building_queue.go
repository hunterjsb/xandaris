package migrations

import (
	"encoding/json"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		json.Unmarshal([]byte(`{
			"id": "building_queue",
			"created": "2024-07-20 10:00:00.000Z",
			"updated": "2024-07-20 10:00:00.000Z",
			"name": "building_queue",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "system_id_field",
					"name": "system_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "systems",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "owner_id_field",
					"name": "owner_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "building_type_field",
					"name": "building_type",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "target_level_field",
					"name": "target_level",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "completion_tick_field",
					"name": "completion_tick",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"noDecimal": true
					}
				}
			],
			"indexes": [
				"CREATE INDEX IF NOT EXISTS idx_system_id ON [[building_queue]] (system_id)",
				"CREATE INDEX IF NOT EXISTS idx_completion_tick ON [[building_queue]] (completion_tick)"
			],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`), collection)

		// Additionally, we need to ensure the "created_at" field is part of the schema.
		// PocketBase automatically adds "created" and "updated" fields, which serve a similar purpose.
		// If a specific "created_at" with game ticks is needed, it should be added as a regular field.
		// For now, we'll rely on the default "created" field.

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		// Optional: Add down migration logic if needed
		// dao := daos.New(db)
		// collection, err := dao.FindCollectionByNameOrId("building_queue")
		// if err != nil {
		// 	return err
		// }
		// return dao.DeleteCollection(collection)
		return nil
	})
}
