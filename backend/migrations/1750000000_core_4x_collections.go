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
		dao := daos.New(db)

		// 1. Create systems collection
		if err := createCollectionFromJSON(dao, `{
			"id": "systems_001",
			"name": "systems",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "name",
					"name": "name",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 100, "pattern": ""}
				},
				{
					"system": false,
					"id": "x",
					"name": "x",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "y",
					"name": "y",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "owner_id",
					"name": "owner_id",
					"type": "relation",
					"required": false,
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
					"id": "richness",
					"name": "richness",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": 10, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_systems_xy ON systems (x, y)", "CREATE INDEX idx_systems_owner ON systems (owner_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 2. Create planets collection
		if err := createCollectionFromJSON(dao, `{
			"id": "planets_001",
			"name": "planets",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "name",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 50, "pattern": ""}
				},
				{
					"system": false,
					"id": "system_id", 
					"name": "system_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "systems",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "planet_type",
					"name": "planet_type",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 30, "pattern": ""}
				},
				{
					"system": false,
					"id": "size",
					"name": "size",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 1, "max": 25, "noDecimal": true}
				},
				{
					"system": false,
					"id": "population",
					"name": "population",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "max_population",
					"name": "max_population",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_planet_system ON planets (system_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 3. Create buildings collection
		if err := createCollectionFromJSON(dao, `{
			"id": "buildings_001",
			"name": "buildings",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "planet_id",
					"name": "planet_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "planets",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "building_type",
					"name": "building_type",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 50, "pattern": ""}
				},
				{
					"system": false,
					"id": "level",
					"name": "level",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": 1, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "active",
					"name": "active",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				},
				{
					"system": false,
					"id": "completion_time",
					"name": "completion_time",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				}
			],
			"indexes": ["CREATE INDEX idx_buildings_planet ON buildings (planet_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 4. Create fleets collection
		if err := createCollectionFromJSON(dao, `{
			"id": "fleets_001",
			"name": "fleets",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "owner_id",
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
					"id": "name",
					"name": "name",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 100, "pattern": ""}
				},
				{
					"system": false,
					"id": "from_id",
					"name": "from_id",
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
					"id": "to_id",
					"name": "to_id",
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
					"id": "eta",
					"name": "eta",
					"type": "date",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				},
				{
					"system": false,
					"id": "eta_tick",
					"name": "eta_tick",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "strength",
					"name": "strength",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": 1, "max": null, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_fleets_owner ON fleets (owner_id)", "CREATE INDEX idx_fleets_eta ON fleets (eta)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 5. Create trade_routes collection
		if err := createCollectionFromJSON(dao, `{
			"id": "trade_routes_001",
			"name": "trade_routes",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "owner_id",
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
					"id": "from_id",
					"name": "from_id",
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
					"id": "to_id",
					"name": "to_id",
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
					"id": "cargo",
					"name": "cargo",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 50, "pattern": ""}
				},
				{
					"system": false,
					"id": "capacity",
					"name": "capacity",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": 1, "max": null, "noDecimal": true}
				},
				{
					"system": false,
					"id": "eta",
					"name": "eta",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				},
				{
					"system": false,
					"id": "eta_tick",
					"name": "eta_tick",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_trade_routes_owner ON trade_routes (owner_id)", "CREATE INDEX idx_trade_routes_eta ON trade_routes (eta)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 6. Create treaties collection
		if err := createCollectionFromJSON(dao, `{
			"id": "treaties_001",
			"name": "treaties",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "type",
					"name": "type",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 50, "pattern": ""}
				},
				{
					"system": false,
					"id": "a_id",
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
					"id": "b_id",
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
					"id": "expires_at",
					"name": "expires_at",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				},
				{
					"system": false,
					"id": "status",
					"name": "status",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 20, "pattern": ""}
				}
			],
			"indexes": ["CREATE INDEX idx_treaties_parties ON treaties (a_id, b_id)", "CREATE INDEX idx_treaties_status ON treaties (status)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Rollback: Drop collections in reverse order
		collections := []string{"treaties", "trade_routes", "fleets", "buildings", "planets", "systems"}
		for _, name := range collections {
			if c, err := dao.FindCollectionByNameOrId(name); err == nil {
				if err := dao.DeleteCollection(c); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func createCollectionFromJSON(dao *daos.Dao, data string) error {
	collection := &models.Collection{}
	if err := json.Unmarshal([]byte(data), collection); err != nil {
		return err
	}
	return dao.SaveCollection(collection)
}