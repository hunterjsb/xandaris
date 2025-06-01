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

		// Helper function to create collections
		createCollection := func(jsonDef string) error {
			collection := &models.Collection{}
			if err := json.Unmarshal([]byte(jsonDef), collection); err != nil {
				return err
			}
			return dao.SaveCollection(collection)
		}

		// 1. Resource Types
		if err := createCollection(`{
			"name": "resource_types",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "description",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "produced_in",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 2. Planet Types
		if err := createCollection(`{
			"name": "planet_types",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "spawn_prob",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 3. Building Types
		if err := createCollection(`{
			"name": "building_types",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "cost",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "strength",
					"type": "select",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": ["strong", "weak", "na"]
					}
				},
				{
					"name": "power_consumption",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "res1_type",
					"type": "relation",
					"required": false,
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
					"name": "res1_quantity",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "res1_capacity",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "res2_type",
					"type": "relation",
					"required": false,
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
					"name": "res2_quantity",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "res2_capacity",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "description",
					"type": "editor",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"convertUrls": false}
				},
				{
					"name": "node_requirement",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 4. Ship Types
		if err := createCollection(`{
			"name": "ship_types",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": true,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "cost",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "strength",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "cargo_capacity",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 5. Systems
		if err := createCollection(`{
			"name": "systems",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "x",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "y",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "discovered_by",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"indexes": ["CREATE UNIQUE INDEX idx_systems_xy ON systems (x, y)"],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 6. Planets
		if err := createCollection(`{
			"name": "planets",
			"type": "base",
			"schema": [
				{
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
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
					"name": "planet_type",
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
				},
				{
					"name": "size",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "colonized_by",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "colonized_at",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				}
			],
			"indexes": ["CREATE INDEX idx_planets_system ON planets (system_id)"],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": "@request.auth.id != ''",
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 7. Resource Nodes
		if err := createCollection(`{
			"name": "resource_nodes",
			"type": "base",
			"schema": [
				{
					"name": "planet_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "planets",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
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
					"name": "richness",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "exhausted",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				}
			],
			"indexes": ["CREATE INDEX idx_resource_nodes_planet ON resource_nodes (planet_id)"],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`); err != nil {
			return err
		}

		// 8. Fleets
		if err := createCollection(`{
			"name": "fleets",
			"type": "base",
			"schema": [
				{
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
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "current_system",
					"type": "relation",
					"required": false,
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
					"name": "destination_system",
					"type": "relation",
					"required": false,
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
					"name": "eta",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				},
				{
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
				}
			],
			"indexes": ["CREATE INDEX idx_fleets_owner ON fleets (owner_id)"],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
			"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"options": {}
		}`); err != nil {
			return err
		}

		// 9. Trade Routes
		if err := createCollection(`{
			"name": "trade_routes",
			"type": "base",
			"schema": [
				{
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
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "pattern": ""}
				},
				{
					"name": "from_system",
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
					"name": "to_system",
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
					"name": "active",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				}
			],
			"indexes": [],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
			"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"options": {}
		}`); err != nil {
			return err
		}

		// 10. Ships
		if err := createCollection(`{
			"name": "ships",
			"type": "base",
			"schema": [
				{
					"name": "fleet_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "fleets",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"name": "ship_type",
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
				},
				{
					"name": "count",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "health",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				}
			],
			"indexes": ["CREATE INDEX idx_ships_fleet ON ships (fleet_id)"],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != ''",
			"updateRule": "@request.auth.id != ''",
			"deleteRule": "@request.auth.id != ''",
			"options": {}
		}`); err != nil {
			return err
		}

		// 11. Populations
		if err := createCollection(`{
			"name": "populations",
			"type": "base",
			"schema": [
				{
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
					"name": "planet_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "planets",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"name": "fleet_id",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "fleets",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"name": "employed_at",
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
					"name": "count",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "happiness",
					"type": "number",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				}
			],
			"indexes": ["CREATE INDEX idx_populations_owner ON populations (owner_id)"],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
			"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			"options": {}
		}`); err != nil {
			return err
		}

		// 12. Buildings (must be last due to dependencies)
		if err := createCollection(`{
			"name": "buildings",
			"type": "base",
			"schema": [
				{
					"name": "planet_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "planets",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"name": "building_type",
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
					"name": "level",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {"min": null, "max": null, "noDecimal": false}
				},
				{
					"name": "active",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				},
				{
					"name": "completion_time",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {"min": "", "max": ""}
				},
				{
					"name": "resource_nodes",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "resource_nodes",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 5,
						"displayFields": null
					}
				}
			],
			"indexes": ["CREATE INDEX idx_buildings_planet ON buildings (planet_id)"],
			"listRule": "@request.auth.id != ''",
			"viewRule": "@request.auth.id != ''",
			"createRule": "@request.auth.id != ''",
			"updateRule": "@request.auth.id != ''",
			"deleteRule": "@request.auth.id != ''",
			"options": {}
		}`); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Drop all collections in reverse order
		collections := []string{
			"buildings", "populations", "ships", "trade_routes", "fleets", 
			"resource_nodes", "planets", "systems", "ship_types", 
			"building_types", "planet_types", "resource_types",
		}

		for _, name := range collections {
			if collection, _ := dao.FindCollectionByNameOrId(name); collection != nil {
				if err := dao.DeleteCollection(collection); err != nil {
					return err
				}
			}
		}

		return nil
	})
}