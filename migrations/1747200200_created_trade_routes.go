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
			"id": "xn_trades_001",
			"created": "2025-01-29 00:02:00.000Z",
			"updated": "2025-01-29 00:02:00.000Z",
			"name": "trade_routes",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "trade_owner",
					"name": "owner_id",
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
					"id": "trade_from",
					"name": "from_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "xn_systems_001",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "trade_to",
					"name": "to_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "xn_systems_001",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "trade_cargo",
					"name": "cargo",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"food",
							"ore",
							"goods",
							"fuel"
						]
					}
				},
				{
					"system": false,
					"id": "trade_cap",
					"name": "cap",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 1,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "trade_eta",
					"name": "eta",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				}
			],
			"indexes": [
				"CREATE INDEX idx_trade_routes_owner ON trade_routes (owner_id)",
				"CREATE INDEX idx_trade_routes_eta ON trade_routes (eta)"
			],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\" && owner_id = @request.auth.id",
			"updateRule": "@request.auth.id != \"\" && owner_id = @request.auth.id",
			"deleteRule": "@request.auth.id != \"\" && owner_id = @request.auth.id",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("xn_trades_001")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}