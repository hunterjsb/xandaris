package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "diplomatic_relations"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()

		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Name:     "player1_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options: &schema.RelationOptions{
					CollectionId: "users",
					MinSelect:    types.Pointer(1),
					MaxSelect:    types.Pointer(1),
				},
			},
			&schema.SchemaField{
				Name:     "player2_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options: &schema.RelationOptions{
					CollectionId: "users",
					MinSelect:    types.Pointer(1),
					MaxSelect:    types.Pointer(1),
				},
			},
			&schema.SchemaField{
				Name:     "status",
				Type:     schema.FieldTypeText,
				Required: true,
				Options:  &schema.TextOptions{},
			},
			&schema.SchemaField{
				Name:     "start_date",
				Type:     schema.FieldTypeDate,
				Required: true,
				Options:  &schema.DateOptions{},
			},
			&schema.SchemaField{
				Name:     "duration_ticks",
				Type:     schema.FieldTypeNumber,
				Required: false,
				Options:  &schema.NumberOptions{},
			},
			&schema.SchemaField{
				Name:     "end_date",
				Type:     schema.FieldTypeDate,
				Required: false,
				Options:  &schema.DateOptions{},
			},
		)

		collection.Indexes = types.JsonArray[string]{
			"CREATE UNIQUE INDEX idx_diplomatic_relations_players ON {{.Name}} (player1_id, player2_id)",
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)
		collection, err := dao.FindCollectionByNameOrId("diplomatic_relations")
		if err != nil {
			return err
		}
		return dao.DeleteCollection(collection)
	})
}
