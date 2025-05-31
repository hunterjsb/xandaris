package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Make game data publicly readable for better UX
		publicReadCollections := map[string]map[string]string{
			// Game world data - publicly readable
			"systems": {
				"listRule": "", // Public read
				"viewRule": "", // Public read
				"createRule": "", // Admin only
				"updateRule": "", // Admin only
				"deleteRule": "", // Admin only
			},
			"planets": {
				"listRule": "", // Public read
				"viewRule": "", // Public read
				"createRule": "", // Admin only
				"updateRule": "@request.auth.id != ''", // Logged in users can colonize
				"deleteRule": "", // Admin only
			},
			"resource_nodes": {
				"listRule": "", // Public read
				"viewRule": "", // Public read
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			// Reference data - publicly readable
			"planet_types": {
				"listRule": "",
				"viewRule": "",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"building_types": {
				"listRule": "",
				"viewRule": "",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"ship_types": {
				"listRule": "",
				"viewRule": "",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			"resource_types": {
				"listRule": "",
				"viewRule": "",
				"createRule": "",
				"updateRule": "",
				"deleteRule": "",
			},
			// Game objects - authenticated users can read, owners can modify
			"buildings": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"fleets": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"ships": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
			"populations": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"trade_routes": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
			},
			"treaties": {
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
			},
		}

		for collectionName, rules := range publicReadCollections {
			collection, err := dao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			// Set list rule
			if listRule, exists := rules["listRule"]; exists {
				if listRule == "" {
					collection.ListRule = nil // Public access
				} else {
					collection.ListRule = &listRule
				}
			}

			// Set view rule
			if viewRule, exists := rules["viewRule"]; exists {
				if viewRule == "" {
					collection.ViewRule = nil // Public access
				} else {
					collection.ViewRule = &viewRule
				}
			}

			// Set create rule
			if createRule, exists := rules["createRule"]; exists {
				if createRule == "" {
					collection.CreateRule = nil // Admin only (no access)
				} else {
					collection.CreateRule = &createRule
				}
			}

			// Set update rule
			if updateRule, exists := rules["updateRule"]; exists {
				if updateRule == "" {
					collection.UpdateRule = nil // Admin only (no access)
				} else {
					collection.UpdateRule = &updateRule
				}
			}

			// Set delete rule
			if deleteRule, exists := rules["deleteRule"]; exists {
				if deleteRule == "" {
					collection.DeleteRule = nil // Admin only (no access)
				} else {
					collection.DeleteRule = &deleteRule
				}
			}

			if err := dao.SaveCollection(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		// Rollback: Set everything back to auth required
		dao := daos.New(db)
		collections := []string{
			"systems", "planets", "resource_nodes", "planet_types", "building_types",
			"ship_types", "resource_types", "buildings", "fleets", "ships",
			"populations", "trade_routes", "treaties",
		}

		for _, collectionName := range collections {
			collection, err := dao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue
			}

			authRequired := "@request.auth.id != ''"
			collection.ListRule = &authRequired
			collection.ViewRule = &authRequired
			collection.CreateRule = &authRequired
			collection.UpdateRule = &authRequired
			collection.DeleteRule = &authRequired

			dao.SaveCollection(collection)
		}

		return nil
	})
}