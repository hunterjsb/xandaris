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
			"id": "xn_treaties_001",
			"created": "2025-01-29 00:03:00.000Z",
			"updated": "2025-01-29 00:03:00.000Z",
			"name": "treaties",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "treaty_type",
					"name": "type",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"alliance",
							"trade_pact",
							"non_aggression",
							"ceasefire"
						]
					}
				},
				{
					"system": false,
					"id": "treaty_a",
					"name": "a_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": ["username"]
					}
				},
				{
					"system": false,
					"id": "treaty_b",
					"name": "b_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": ["username"]
					}
				},
				{
					"system": false,
					"id": "treaty_created",
					"name": "created_at",
					"type": "date",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				},
				{
					"system": false,
					"id": "treaty_expires",
					"name": "expires_at",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				},
				{
					"system": false,
					"id": "treaty_status",
					"name": "status",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"proposed",
							"active",
							"expired",
							"broken"
						]
					}
				}
			],
			"indexes": [
				"CREATE INDEX idx_treaties_players ON treaties (a_id, b_id)",
				"CREATE INDEX idx_treaties_status ON treaties (status)"
			],
			"listRule": "@request.auth.id != \"\" && (a_id = @request.auth.id || b_id = @request.auth.id)",
			"viewRule": "@request.auth.id != \"\" && (a_id = @request.auth.id || b_id = @request.auth.id)",
			"createRule": "@request.auth.id != \"\" && a_id = @request.auth.id",
			"updateRule": "@request.auth.id != \"\" && (a_id = @request.auth.id || b_id = @request.auth.id)",
			"deleteRule": "@request.auth.id != \"\" && (a_id = @request.auth.id || b_id = @request.auth.id)",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("xn_treaties_001")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}