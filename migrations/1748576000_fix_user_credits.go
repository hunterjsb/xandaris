package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Remove credits_per_tick field
		for i := len(collection.Schema.Fields()) - 1; i >= 0; i-- {
			if collection.Schema.Fields()[i].Name == "credits_per_tick" {
				collection.Schema.RemoveField(collection.Schema.Fields()[i].Id)
				break
			}
		}

		// Add credits field
		new_credits := &schema.SchemaField{}
		json.Unmarshal([]byte(`{
			"system": false,
			"id": "credits",
			"name": "credits",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": true
			}
		}`), new_credits)
		collection.Schema.AddField(new_credits)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Remove credits field
		for i := len(collection.Schema.Fields()) - 1; i >= 0; i-- {
			if collection.Schema.Fields()[i].Name == "credits" {
				collection.Schema.RemoveField(collection.Schema.Fields()[i].Id)
				break
			}
		}

		// Add back credits_per_tick field
		new_credits_per_tick := &schema.SchemaField{}
		json.Unmarshal([]byte(`{
			"system": false,
			"id": "credits_per_tick",
			"name": "credits_per_tick",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": true
			}
		}`), new_credits_per_tick)
		collection.Schema.AddField(new_credits_per_tick)

		return dao.SaveCollection(collection)
	})
}