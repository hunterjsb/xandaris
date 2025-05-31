package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("systems")
		if err != nil {
			return err
		}

		// Set rules to null for public read access
		collection.ListRule = nil
		collection.ViewRule = nil

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("systems")
		if err != nil {
			return err
		}

		// Revert to authenticated access
		listRule := ""
		viewRule := ""
		collection.ListRule = &listRule
		collection.ViewRule = &viewRule

		return dao.SaveCollection(collection)
	})
}