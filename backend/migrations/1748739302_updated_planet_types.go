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

		// remove
		collection.Schema.RemoveField("fdymn2xc")

		// remove
		collection.Schema.RemoveField("uu3kyf66")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("qefrhy1j9sbwn6l")
		if err != nil {
			return err
		}

		// add
		del_base_max_population := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fdymn2xc",
			"name": "base_max_population",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), del_base_max_population); err != nil {
			return err
		}
		collection.Schema.AddField(del_base_max_population)

		// add
		del_habitability := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "uu3kyf66",
			"name": "habitability",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), del_habitability); err != nil {
			return err
		}
		collection.Schema.AddField(del_habitability)

		return dao.SaveCollection(collection)
	})
}
