package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		emptyRule := ""
		maxSelectOne := 1

		collection := &models.Collection{}
		collection.Name = "hyperlanes"
		collection.Type = "base"
		collection.ListRule = &emptyRule
		collection.ViewRule = &emptyRule
		
		// from_system field
		fromSystemField := &schema.SchemaField{
			Name:     "from_system",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "systems",
				CascadeDelete: false,
				MinSelect:     nil,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		}
		collection.Schema.AddField(fromSystemField)

		// to_system field
		toSystemField := &schema.SchemaField{
			Name:     "to_system",
			Type:     schema.FieldTypeRelation,
			Required: true,
			Options: &schema.RelationOptions{
				CollectionId:  "systems",
				CascadeDelete: false,
				MinSelect:     nil,
				MaxSelect:     &maxSelectOne,
				DisplayFields: nil,
			},
		}
		collection.Schema.AddField(toSystemField)

		// distance field
		distanceField := &schema.SchemaField{
			Name:     "distance",
			Type:     schema.FieldTypeNumber,
			Required: true,
			Options: &schema.NumberOptions{
				Min: nil,
				Max: nil,
			},
		}
		collection.Schema.AddField(distanceField)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("hyperlanes")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}