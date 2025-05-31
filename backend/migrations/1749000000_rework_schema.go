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

		// modify systems collection - remove old resource and building fields
		systems, err := dao.FindCollectionByNameOrId("systems")
		if err == nil {
			systems.Schema.RemoveField("sys_pop")
			systems.Schema.RemoveField("sys_morale")
			systems.Schema.RemoveField("sys_food")
			systems.Schema.RemoveField("sys_ore")
			systems.Schema.RemoveField("sys_goods")
			systems.Schema.RemoveField("sys_fuel")
			systems.Schema.RemoveField("sys_hab_lvl")
			systems.Schema.RemoveField("sys_farm_lvl")
			systems.Schema.RemoveField("sys_mine_lvl")
			systems.Schema.RemoveField("sys_fac_lvl")
			systems.Schema.RemoveField("sys_yard_lvl")
			if err := dao.SaveCollection(systems); err != nil {
				return err
			}
		}

		// add trade_route_id field to fleets
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

		// create planet_types collection
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
                    "options": {
                        "min": null,
                        "max": 50,
                        "pattern": ""
                    }
                }
            ],
            "indexes": [],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`); err != nil {
			return err
		}

		// create planets collection
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
                    "options": {
                        "min": null,
                        "max": 50,
                        "pattern": ""
                    }
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
            "indexes": [
                "CREATE INDEX idx_planet_system ON planets (system_id)"
            ],
            "listRule": "",
            "viewRule": "",
            "createRule": "@request.auth.id != \"\"",
            "updateRule": "@request.auth.id != \"\"",
            "deleteRule": "@request.auth.id != \"\"",
            "options": {}
        }`); err != nil {
			return err
		}

		// create building_types collection
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
                    "options": {
                        "min": null,
                        "max": 50,
                        "pattern": ""
                    }
                }
            ],
            "indexes": [],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`); err != nil {
			return err
		}

		// create buildings collection
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
                    "options": {
                        "min": 1,
                        "max": null,
                        "noDecimal": true
                    }
                }
            ],
            "indexes": [
                "CREATE INDEX idx_buildings_planet ON buildings (planet_id)"
            ],
            "listRule": "",
            "viewRule": "",
            "createRule": "@request.auth.id != \"\"",
            "updateRule": "@request.auth.id != \"\"",
            "deleteRule": "@request.auth.id != \"\"",
            "options": {}
        }`); err != nil {
			return err
		}

		// create resource_types collection
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
                    "options": {
                        "min": null,
                        "max": 50,
                        "pattern": ""
                    }
                }
            ],
            "indexes": [],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`); err != nil {
			return err
		}

		// create resource_nodes collection
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
                    "options": {
                        "min": 0,
                        "max": null,
                        "noDecimal": true
                    }
                }
            ],
            "indexes": [
                "CREATE INDEX idx_resource_nodes_planet ON resource_nodes (planet_id)"
            ],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`); err != nil {
			return err
		}

		// create ship_types collection
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
                    "options": {
                        "min": null,
                        "max": 50,
                        "pattern": ""
                    }
                }
            ],
            "indexes": [],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`); err != nil {
			return err
		}

		// create ships collection
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
            "indexes": [
                "CREATE INDEX idx_ships_fleet ON ships (fleet_id)"
            ],
            "listRule": "",
            "viewRule": "",
            "createRule": "@request.auth.id != \"\"",
            "updateRule": "@request.auth.id != \"\"",
            "deleteRule": "@request.auth.id != \"\"",
            "options": {}
        }`); err != nil {
			return err
		}

		// create populations collection
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
                    "options": {
                        "min": 0,
                        "max": null,
                        "noDecimal": true
                    }
                }
            ],
            "indexes": [
                "CREATE INDEX idx_pop_user ON populations (user_id)"
            ],
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

		// rollback: remove added field from fleets
		if fleets, err := dao.FindCollectionByNameOrId("fleets"); err == nil {
			fleets.Schema.RemoveField("trade_route_id")
			if err := dao.SaveCollection(fleets); err != nil {
				return err
			}
		}

		// rollback: drop new collections
		for _, name := range []string{
			"populations",
			"ships",
			"ship_types",
			"resource_nodes",
			"resource_types",
			"buildings",
			"building_types",
			"planets",
			"planet_types",
		} {
			if c, err := dao.FindCollectionByNameOrId(name); err == nil {
				if err := dao.DeleteCollection(c); err != nil {
					return err
				}
			}
		}

		// rollback: add old fields back to systems
		if systems, err := dao.FindCollectionByNameOrId("systems"); err == nil {
			// pop
			field := &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_pop",
                "name": "pop",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":null,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// morale
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_morale",
                "name": "morale",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":100,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// food
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_food",
                "name": "food",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":null,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// ore
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_ore",
                "name": "ore",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":null,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// goods
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_goods",
                "name": "goods",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":null,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// fuel
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_fuel",
                "name": "fuel",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":null,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// hab_lvl
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_hab_lvl",
                "name": "hab_lvl",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":10,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// farm_lvl
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_farm_lvl",
                "name": "farm_lvl",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":10,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// mine_lvl
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_mine_lvl",
                "name": "mine_lvl",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":10,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// fac_lvl
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_fac_lvl",
                "name": "fac_lvl",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":10,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)
			// yard_lvl
			field = &schema.SchemaField{}
			json.Unmarshal([]byte(`{
                "system": false,
                "id": "sys_yard_lvl",
                "name": "yard_lvl",
                "type": "number",
                "required": false,
                "presentable": false,
                "unique": false,
                "options": {"min":0,"max":10,"noDecimal":true}
            }`), field)
			systems.Schema.AddField(field)

			if err := dao.SaveCollection(systems); err != nil {
				return err
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
