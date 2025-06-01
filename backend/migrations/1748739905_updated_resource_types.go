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

		collection, err := dao.FindCollectionByNameOrId("uu9p0hpnvum6j8g")
		if err != nil {
			return err
		}

		// update
		edit_is_raw := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "lumilgs4",
			"name": "is_raw",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), edit_is_raw); err != nil {
			return err
		}
		collection.Schema.AddField(edit_is_raw)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("uu9p0hpnvum6j8g")
		if err != nil {
			return err
		}

		// update
		edit_is_raw := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "lumilgs4",
			"name": "is_consumable",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), edit_is_raw); err != nil {
			return err
		}
		collection.Schema.AddField(edit_is_raw)

		return dao.SaveCollection(collection)
	})
}
