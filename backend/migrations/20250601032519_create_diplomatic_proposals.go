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

		collection.Name = "diplomatic_proposals"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()

		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Name:     "proposer_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options: &schema.RelationOptions{
					CollectionId: "users",
					MinSelect:    types.Pointer(1),
					MaxSelect:    types.Pointer(1),
				},
			},
			&schema.SchemaField{
				Name:     "receiver_id",
				Type:     schema.FieldTypeRelation,
				Required: true,
				Options: &schema.RelationOptions{
					CollectionId: "users",
					MinSelect:    types.Pointer(1),
					MaxSelect:    types.Pointer(1),
				},
			},
			&schema.SchemaField{
				Name:     "type",
				Type:     schema.FieldTypeText,
				Required: true,
				Options:  &schema.TextOptions{},
			},
			&schema.SchemaField{
				Name:     "terms",
				Type:     schema.FieldTypeJson,
				Required: false,
				Options:  &schema.JsonOptions{},
			},
			&schema.SchemaField{
				Name:     "status",
				Type:     schema.FieldTypeText,
				Required: true,
				Options:  &schema.TextOptions{},
			},
			&schema.SchemaField{
				Name:     "proposed_date",
				Type:     schema.FieldTypeDate,
				Required: true,
				Options:  &schema.DateOptions{},
			},
			&schema.SchemaField{
				Name:     "expiration_date",
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
		)

		// Example of a non-unique index (if needed, e.g., for proposal status and receiver)
		// collection.Indexes = types.JsonArray[string]{
		//  "CREATE INDEX idx_diplomatic_proposals_status_receiver ON {{.Name}} (status, receiver_id)",
		// }

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)
		collection, err := dao.FindCollectionByNameOrId("diplomatic_proposals")
		if err != nil {
			return err
		}
		return dao.DeleteCollection(collection)
	})
}
