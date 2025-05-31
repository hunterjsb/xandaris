package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// First, drop any existing collections that might conflict
		existingCollections := []string{
			"buildings", "population", "resource_nodes", "building_types", 
			"planet_types", "resource_types", "ship_types", "planets", "ships", "populations",
		}
		for _, name := range existingCollections {
			if c, err := dao.FindCollectionByNameOrId(name); err == nil {
				dao.DeleteCollection(c)
			}
		}

		// 1. Create planet_types collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_planet_types_001",
			"name": "planet_types", 
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "pt_name",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": 50, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 2. Create resource_types collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_resource_types_001",
			"name": "resource_types",
			"type": "base", 
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "rt_name",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": 50, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 3. Create building_types collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_building_types_001",
			"name": "building_types",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "bt_name", 
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": 50, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 4. Create ship_types collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_ship_types_001",
			"name": "ship_types",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "st_name",
					"name": "name", 
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": 50, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 5. Create planets collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_planets_001",
			"name": "planets",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "pl_name",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": 50, "pattern": ""}
				},
				{
					"system": false,
					"id": "pl_system",
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
					"id": "pl_type",
					"name": "type_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "planet_types",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
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

		// 6. Create buildings collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_buildings_001",
			"name": "buildings",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "b_planet",
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
					"id": "b_type",
					"name": "type_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "building_types",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "b_level",
					"name": "level",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": 1, "max": null, "noDecimal": true}
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

		// 7. Create resource_nodes collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_resource_nodes_001",
			"name": "resource_nodes",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "rn_planet",
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
					"id": "rn_type",
					"name": "type_id",
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
					"id": "rn_building",
					"name": "building_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "buildings",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "rn_richness",
					"name": "richness",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_resource_nodes_planet ON resource_nodes (planet_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 8. Create ships collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_ships_001",
			"name": "ships",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "s_fleet",
					"name": "fleet_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "fleets",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "s_type",
					"name": "type_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "ship_types",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"indexes": ["CREATE INDEX idx_ships_fleet ON ships (fleet_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 9. Create populations collection
		if err := createCollectionFromJSON(dao, `{
			"id": "xn_populations_001",
			"name": "populations",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "pop_user",
					"name": "user_id",
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
					"id": "pop_planet",
					"name": "planet_id",
					"type": "relation",
					"required": false,
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
					"id": "pop_fleet",
					"name": "fleet_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "fleets",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "pop_building",
					"name": "building_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "buildings",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "pop_size",
					"name": "size",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": 0, "max": null, "noDecimal": true}
				}
			],
			"indexes": ["CREATE INDEX idx_pop_user ON populations (user_id)"],
			"listRule": "",
			"viewRule": "",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id != \"\"",
			"deleteRule": "@request.auth.id != \"\"",
			"options": {}
		}`); err != nil {
			return err
		}

		// 10. Add trade_route_id field to fleets
		fleets, err := dao.FindCollectionByNameOrId("fleets")
		if err == nil {
			newField := &schema.SchemaField{}
			if err := json.Unmarshal([]byte(`{
				"system": false,
				"id": "trade_route_id",
				"name": "trade_route_id",
				"type": "relation",
				"required": false,
				"presentable": false,
				"unique": false,
				"options": {
					"collectionId": "trade_routes",
					"cascadeDelete": false,
					"minSelect": null,
					"maxSelect": 1,
					"displayFields": null
				}
			}`), newField); err != nil {
				return err
			}
			fleets.Schema.AddField(newField)
			if err := dao.SaveCollection(fleets); err != nil {
				return err
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Rollback: Drop new collections in reverse order
		collections := []string{
			"populations", "ships", "resource_nodes", "buildings", 
			"planets", "ship_types", "building_types", "resource_types", "planet_types",
		}
		for _, name := range collections {
			if c, err := dao.FindCollectionByNameOrId(name); err == nil {
				if err := dao.DeleteCollection(c); err != nil {
					return err
				}
			}
		}

		// Rollback: Remove trade_route_id from fleets
		if fleets, err := dao.FindCollectionByNameOrId("fleets"); err == nil {
			fleets.Schema.RemoveField("trade_route_id")
			dao.SaveCollection(fleets)
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