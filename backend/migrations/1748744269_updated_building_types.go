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
		collection.Schema.RemoveField("cgbioj6d")

		// remove
		collection.Schema.RemoveField("bu2f4vpf")

		// add
		new_res1_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "cpar1wpl",
			"name": "res1_type",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "uu9p0hpnvum6j8g",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_res1_type); err != nil {
			return err
		}
		collection.Schema.AddField(new_res1_type)

		// add
		new_res2_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "kwxxgxzr",
			"name": "res2_type",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "uu9p0hpnvum6j8g",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_res2_type); err != nil {
			return err
		}
		collection.Schema.AddField(new_res2_type)

		// add
		new_node_requirement := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tr2vsh4v",
			"name": "node_requirement",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_node_requirement); err != nil {
			return err
		}
		collection.Schema.AddField(new_node_requirement)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cz4eps74xi3j7m7")
		if err != nil {
			return err
		}

		// add
		del_res1_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "cgbioj6d",
			"name": "res1_type",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_res1_type); err != nil {
			return err
		}
		collection.Schema.AddField(del_res1_type)

		// add
		del_res2_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "bu2f4vpf",
			"name": "res2_type",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_res2_type); err != nil {
			return err
		}
		collection.Schema.AddField(del_res2_type)

		// remove
		collection.Schema.RemoveField("cpar1wpl")

		// remove
		collection.Schema.RemoveField("kwxxgxzr")

		// remove
		collection.Schema.RemoveField("tr2vsh4v")

		return dao.SaveCollection(collection)
	})
}
