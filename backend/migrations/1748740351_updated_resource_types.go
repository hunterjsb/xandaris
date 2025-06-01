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

		// remove
		collection.Schema.RemoveField("lumilgs4")

		// add
		new_produced_in := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tnoraq2p",
			"name": "produced_in",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_produced_in); err != nil {
			return err
		}
		collection.Schema.AddField(new_produced_in)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("uu9p0hpnvum6j8g")
		if err != nil {
			return err
		}

		// add
		del_is_raw := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "lumilgs4",
			"name": "is_raw",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_is_raw); err != nil {
			return err
		}
		collection.Schema.AddField(del_is_raw)

		// remove
		collection.Schema.RemoveField("tnoraq2p")

		return dao.SaveCollection(collection)
	})
}
