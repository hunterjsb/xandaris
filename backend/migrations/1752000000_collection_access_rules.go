package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Update collections with proper access rules
		collections := map[string]map[string]string{
			"fleets": {
				"listRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"viewRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"buildings": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"trade_routes": {
				"listRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"viewRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"treaties": {
				"listRule":   "@request.auth.id != '' && (player_a = @request.auth.id || player_b = @request.auth.id)",
				"viewRule":   "@request.auth.id != '' && (player_a = @request.auth.id || player_b = @request.auth.id)",
				"createRule": "@request.auth.id != '' && (@request.data.player_a = @request.auth.id || @request.data.player_b = @request.auth.id)",
				"updateRule": "@request.auth.id != '' && (player_a = @request.auth.id || player_b = @request.auth.id)",
				"deleteRule": "@request.auth.id != '' && (player_a = @request.auth.id || player_b = @request.auth.id)",
			},
			"populations": {
				"listRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"viewRule":   "@request.auth.id != '' && owner_id = @request.auth.id",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"ships": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"planets": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"systems": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"resource_nodes": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"resource_types": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"planet_types": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"building_types": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"ship_types": {
				"listRule":   "@request.auth.id != ''",
				"viewRule":   "@request.auth.id != ''",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
		}

		for collectionName, rules := range collections {
			collection, err := dao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			if rules["listRule"] != "" {
				listRule := rules["listRule"]
				collection.ListRule = &listRule
			}
			if rules["viewRule"] != "" {
				viewRule := rules["viewRule"]
				collection.ViewRule = &viewRule
			}
			if rules["createRule"] != "" {
				createRule := rules["createRule"]
				collection.CreateRule = &createRule
			} else if rules["createRule"] == "" {
				collection.CreateRule = nil
			}
			if rules["updateRule"] != "" {
				updateRule := rules["updateRule"]
				collection.UpdateRule = &updateRule
			} else if rules["updateRule"] == "" {
				collection.UpdateRule = nil
			}
			if rules["deleteRule"] != "" {
				deleteRule := rules["deleteRule"]
				collection.DeleteRule = &deleteRule
			} else if rules["deleteRule"] == "" {
				collection.DeleteRule = nil
			}

			if err := dao.SaveCollection(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		// Rollback: Remove all access rules
		dao := daos.New(db)
		collections := []string{
			"fleets", "buildings", "trade_routes", "treaties", "populations",
			"ships", "planets", "systems", "resource_nodes", "resource_types",
			"planet_types", "building_types", "ship_types",
		}

		for _, collectionName := range collections {
			collection, err := dao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue
			}

			collection.ListRule = nil
			collection.ViewRule = nil
			collection.CreateRule = nil
			collection.UpdateRule = nil
			collection.DeleteRule = nil

			dao.SaveCollection(collection)
		}

		return nil
	})
}