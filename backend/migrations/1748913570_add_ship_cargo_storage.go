package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "ship_cargo",
			"created": "2025-06-03 01:15:00.000Z",
			"updated": "2025-06-03 01:15:00.000Z",
			"name": "ship_cargo",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "ship_id",
					"name": "ship_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "ships",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "resource_type",
					"name": "resource_type",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "resource_types",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "quantity",
					"name": "quantity",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": false
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX idx_ship_cargo_unique ON ship_cargo (ship_id, resource_type)",
				"CREATE INDEX idx_ship_cargo_ship ON ship_cargo (ship_id)"
			],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != ''",
			"updateRule": "@request.auth.id != ''",
			"deleteRule": "@request.auth.id != ''",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("ship_cargo")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}