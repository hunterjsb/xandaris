package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Update resource_nodes collection access rules
		resourceNodesCollection, err := dao.FindCollectionByNameOrId("resource_nodes")
		if err != nil {
			return err
		}

		emptyRule := ""
		resourceNodesCollection.ListRule = &emptyRule
		resourceNodesCollection.ViewRule = &emptyRule

		if err := dao.SaveCollection(resourceNodesCollection); err != nil {
			return err
		}

		// Update resource_types collection access rules
		resourceTypesCollection, err := dao.FindCollectionByNameOrId("resource_types")
		if err != nil {
			return err
		}

		resourceTypesCollection.ListRule = &emptyRule
		resourceTypesCollection.ViewRule = &emptyRule

		return dao.SaveCollection(resourceTypesCollection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Revert resource_nodes collection to admin-only access
		resourceNodesCollection, err := dao.FindCollectionByNameOrId("resource_nodes")
		if err != nil {
			return err
		}

		resourceNodesCollection.ListRule = nil
		resourceNodesCollection.ViewRule = nil

		if err := dao.SaveCollection(resourceNodesCollection); err != nil {
			return err
		}

		// Revert resource_types collection to admin-only access
		resourceTypesCollection, err := dao.FindCollectionByNameOrId("resource_types")
		if err != nil {
			return err
		}

		resourceTypesCollection.ListRule = nil
		resourceTypesCollection.ViewRule = nil

		return dao.SaveCollection(resourceTypesCollection)
	})
}