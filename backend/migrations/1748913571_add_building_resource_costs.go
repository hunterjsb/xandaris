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

		collection, err := dao.FindCollectionByNameOrId("building_types")
		if err != nil {
			return err
		}

		// Add cost_resource_type field
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "cost_resource_type",
			Type:     schema.FieldTypeRelation,
			Required: false,
			Options: &schema.RelationOptions{
				CollectionId:  "resource_types",
				CascadeDelete: false,
				MinSelect:     nil,
				MaxSelect:     func() *int { i := 1; return &i }(),
				DisplayFields: nil,
			},
		})

		// Add cost_quantity field (rename existing cost field)
		collection.Schema.AddField(&schema.SchemaField{
			Name:     "cost_quantity",
			Type:     schema.FieldTypeNumber,
			Required: false,
			Options: &schema.NumberOptions{
				Min:       func() *float64 { f := 0.0; return &f }(),
				Max:       nil,
				NoDecimal: false,
			},
		})

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("building_types")
		if err != nil {
			return err
		}

		// Remove the fields we added
		collection.Schema.RemoveField("cost_resource_type")
		collection.Schema.RemoveField("cost_quantity")

		return dao.SaveCollection(collection)
	})
}