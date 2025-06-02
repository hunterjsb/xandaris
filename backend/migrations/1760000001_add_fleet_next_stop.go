package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("fleets")
		if err != nil {
			return err
		}

		// Add next_stop field as a relation to systems
		maxSelect := 1
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "next_stop",
			Type:     schema.FieldTypeRelation,
			Required: false,
			Options: &schema.RelationOptions{
				CollectionId:  "systems",
				CascadeDelete: false,
				MinSelect:     nil,
				MaxSelect:     &maxSelect,
				DisplayFields: nil,
			},
		})

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("fleets")
		if err != nil {
			return err
		}

		// Remove next_stop field
		collection.Schema.RemoveField("next_stop")

		return dao.SaveCollection(collection)
	})
}