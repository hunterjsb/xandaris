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

		collection, err := dao.FindCollectionByNameOrId("cz4eps74xi3j7m7")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("q3k4yjmj")

		// remove
		collection.Schema.RemoveField("zgl369mt")

		// add
		new_strength := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "smaffnc2",
			"name": "strength",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"strong",
					"week",
					"na"
				]
			}
		}`), new_strength); err != nil {
			return err
		}
		collection.Schema.AddField(new_strength)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cz4eps74xi3j7m7")
		if err != nil {
			return err
		}

		// add
		del_worker_capacity := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "q3k4yjmj",
			"name": "worker_capacity",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), del_worker_capacity); err != nil {
			return err
		}
		collection.Schema.AddField(del_worker_capacity)

		// add
		del_max_level := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zgl369mt",
			"name": "max_level",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), del_max_level); err != nil {
			return err
		}
		collection.Schema.AddField(del_max_level)

		// remove
		collection.Schema.RemoveField("smaffnc2")

		return dao.SaveCollection(collection)
	})
}
