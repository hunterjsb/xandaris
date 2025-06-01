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

		collection, err := dao.FindCollectionByNameOrId("qefrhy1j9sbwn6l")
		if err != nil {
			return err
		}

		// add
		new_spawn_prob := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "3poxue6r",
			"name": "spawn_prob",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_spawn_prob); err != nil {
			return err
		}
		collection.Schema.AddField(new_spawn_prob)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("qefrhy1j9sbwn6l")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("3poxue6r")

		return dao.SaveCollection(collection)
	})
}
