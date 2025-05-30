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
			"id": "xn_systems_001",
			"created": "2025-01-29 00:00:00.000Z",
			"updated": "2025-01-29 00:00:00.000Z",
			"name": "systems",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "sys_x",
					"name": "x",
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
					"id": "sys_y",
					"name": "y",
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
					"id": "sys_richness",
					"name": "richness",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_owner",
					"name": "owner_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": ["username"]
					}
				},
				{
					"system": false,
					"id": "sys_pop",
					"name": "pop",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_morale",
					"name": "morale",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 100,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_food",
					"name": "food",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_ore",
					"name": "ore",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_goods",
					"name": "goods",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_fuel",
					"name": "fuel",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_hab_lvl",
					"name": "hab_lvl",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_farm_lvl",
					"name": "farm_lvl",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_mine_lvl",
					"name": "mine_lvl",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_fac_lvl",
					"name": "fac_lvl",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				},
				{
					"system": false,
					"id": "sys_yard_lvl",
					"name": "yard_lvl",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": 10,
						"noDecimal": true
					}
				}
			],
			"indexes": [
				"CREATE INDEX idx_systems_coords ON systems (x, y)",
				"CREATE INDEX idx_systems_owner ON systems (owner_id)"
			],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\" && owner_id = @request.auth.id",
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("xn_systems_001")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}