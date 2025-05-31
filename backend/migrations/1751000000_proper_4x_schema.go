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

		// Drop old collections first
		collections := []string{"systems_001", "planets_001", "buildings_001", "fleets_001", "trade_routes_001"}
		for _, collectionName := range collections {
			if collection, _ := dao.FindCollectionByNameOrId(collectionName); collection != nil {
				if err := dao.DeleteCollection(collection); err != nil {
					// Continue if collection doesn't exist
				}
			}
		}

		// 1. Resource Types
		if err := createCollectionFromJSON(dao, `{
			"name": "resource_types",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text", "required": true, "unique": true},
				{"name": "description", "type": "text"},
				{"name": "is_consumable", "type": "bool"}
			]
		}`); err != nil {
			return err
		}

		// 2. Planet Types
		if err := createCollectionFromJSON(dao, `{
			"name": "planet_types",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text", "required": true, "unique": true},
				{"name": "base_max_population", "type": "number", "required": true},
				{"name": "habitability", "type": "number", "required": true}
			]
		}`); err != nil {
			return err
		}

		// 3. Building Types
		if err := createCollectionFromJSON(dao, `{
			"name": "building_types",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text", "required": true, "unique": true},
				{"name": "cost", "type": "number", "required": true},
				{"name": "worker_capacity", "type": "number", "required": true},
				{"name": "max_level", "type": "number", "required": true}
			]
		}`); err != nil {
			return err
		}

		// 4. Ship Types
		if err := createCollectionFromJSON(dao, `{
			"name": "ship_types",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text", "required": true, "unique": true},
				{"name": "cost", "type": "number", "required": true},
				{"name": "strength", "type": "number", "required": true},
				{"name": "cargo_capacity", "type": "number"}
			]
		}`); err != nil {
			return err
		}

		// 5. Systems
		if err := createCollectionFromJSON(dao, `{
			"name": "systems",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text"},
				{"name": "x", "type": "number", "required": true},
				{"name": "y", "type": "number", "required": true},
				{"name": "discovered_by", "type": "relation", "options": {"collectionId": "_pb_users_auth_"}}
			],
			"indexes": ["CREATE UNIQUE INDEX idx_systems_xy ON systems (x, y)"]
		}`); err != nil {
			return err
		}

		// 6. Planets
		if err := createCollectionFromJSON(dao, `{
			"name": "planets",
			"type": "base",
			"schema": [
				{"name": "name", "type": "text", "required": true},
				{"name": "system_id", "type": "relation", "required": true, "options": {"collectionId": "systems"}},
				{"name": "planet_type", "type": "relation", "required": true, "options": {"collectionId": "planet_types"}},
				{"name": "size", "type": "number", "required": true},
				{"name": "colonized_by", "type": "text"},
				{"name": "colonized_at", "type": "date"}
			],
			"indexes": ["CREATE INDEX idx_planets_system ON planets (system_id)"]
		}`); err != nil {
			return err
		}

		// 7. Resource Nodes
		if err := createCollectionFromJSON(dao, `{
			"name": "resource_nodes",
			"type": "base",
			"schema": [
				{"name": "planet_id", "type": "relation", "required": true, "options": {"collectionId": "planets"}},
				{"name": "resource_type", "type": "relation", "required": true, "options": {"collectionId": "resource_types"}},
				{"name": "richness", "type": "number", "required": true},
				{"name": "exhausted", "type": "bool"}
			],
			"indexes": ["CREATE INDEX idx_resource_nodes_planet ON resource_nodes (planet_id)"]
		}`); err != nil {
			return err
		}

		// 8. Buildings
		if err := createCollectionFromJSON(dao, `{
			"name": "buildings",
			"type": "base",
			"schema": [
				{"name": "planet_id", "type": "relation", "required": true, "options": {"collectionId": "planets"}},
				{"name": "building_type", "type": "relation", "required": true, "options": {"collectionId": "building_types"}},
				{"name": "level", "type": "number", "required": true},
				{"name": "active", "type": "bool"},
				{"name": "completion_time", "type": "date"},
				{"name": "resource_nodes", "type": "relation", "options": {"collectionId": "resource_nodes", "maxSelect": 5}}
			],
			"indexes": ["CREATE INDEX idx_buildings_planet ON buildings (planet_id)"]
		}`); err != nil {
			return err
		}

		// 9. Populations
		if err := createCollectionFromJSON(dao, `{
			"name": "populations",
			"type": "base",
			"schema": [
				{"name": "owner_id", "type": "relation", "required": true, "options": {"collectionId": "_pb_users_auth_"}},
				{"name": "planet_id", "type": "relation", "options": {"collectionId": "planets"}},
				{"name": "fleet_id", "type": "relation", "options": {"collectionId": "fleets"}},
				{"name": "employed_at", "type": "relation", "options": {"collectionId": "buildings"}},
				{"name": "count", "type": "number", "required": true},
				{"name": "happiness", "type": "number"}
			],
			"indexes": ["CREATE INDEX idx_populations_owner ON populations (owner_id)"]
		}`); err != nil {
			return err
		}

		// 10. Fleets
		if err := createCollectionFromJSON(dao, `{
			"name": "fleets",
			"type": "base",
			"schema": [
				{"name": "owner_id", "type": "relation", "required": true, "options": {"collectionId": "_pb_users_auth_"}},
				{"name": "name", "type": "text", "required": true},
				{"name": "current_system", "type": "relation", "options": {"collectionId": "systems"}},
				{"name": "destination_system", "type": "relation", "options": {"collectionId": "systems"}},
				{"name": "eta", "type": "date"},
				{"name": "trade_route_id", "type": "relation", "options": {"collectionId": "trade_routes"}}
			],
			"indexes": ["CREATE INDEX idx_fleets_owner ON fleets (owner_id)"]
		}`); err != nil {
			return err
		}

		// 11. Ships
		if err := createCollectionFromJSON(dao, `{
			"name": "ships",
			"type": "base",
			"schema": [
				{"name": "fleet_id", "type": "relation", "required": true, "options": {"collectionId": "fleets"}},
				{"name": "ship_type", "type": "relation", "required": true, "options": {"collectionId": "ship_types"}},
				{"name": "count", "type": "number", "required": true},
				{"name": "health", "type": "number"}
			],
			"indexes": ["CREATE INDEX idx_ships_fleet ON ships (fleet_id)"]
		}`); err != nil {
			return err
		}

		// 12. Trade Routes
		if err := createCollectionFromJSON(dao, `{
			"name": "trade_routes",
			"type": "base",
			"schema": [
				{"name": "owner_id", "type": "relation", "required": true, "options": {"collectionId": "_pb_users_auth_"}},
				{"name": "name", "type": "text", "required": true},
				{"name": "from_system", "type": "relation", "required": true, "options": {"collectionId": "systems"}},
				{"name": "to_system", "type": "relation", "required": true, "options": {"collectionId": "systems"}},
				{"name": "resource_type", "type": "relation", "required": true, "options": {"collectionId": "resource_types"}},
				{"name": "active", "type": "bool"}
			],
			"indexes": ["CREATE INDEX idx_trade_routes_owner ON trade_routes (owner_id)"]
		}`); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		// Rollback: drop all new collections
		dao := daos.New(db)
		collections := []string{
			"resource_types", "planet_types", "building_types", "ship_types",
			"systems", "planets", "resource_nodes", "buildings", "populations",
			"fleets", "ships", "trade_routes",
		}
		for _, name := range collections {
			if collection, _ := dao.FindCollectionByNameOrId(name); collection != nil {
				dao.DeleteCollection(collection)
			}
		}
		return nil
	})
}

func createCollectionFromJSON(dao *daos.Dao, jsonData string) error {
	collection := &models.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return err
	}
	return dao.SaveCollection(collection)
}

