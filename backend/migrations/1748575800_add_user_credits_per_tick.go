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

		// Get users collection
		collection, err := dao.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Add credits_per_tick field for banking income
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "credits_per_tick",
			Type:     schema.FieldTypeNumber,
			Required: false,
			Options: &schema.NumberOptions{
				Min: &[]float64{0}[0],
			},
		})

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Get users collection
		collection, err := dao.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Remove credits_per_tick field
		collection.Schema.RemoveField("credits_per_tick")

		return dao.SaveCollection(collection)
	})
}