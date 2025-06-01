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

		// add
		new_res1_type := &schema.SchemaField{}
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
		}`), new_res1_type); err != nil {
			return err
		}
		collection.Schema.AddField(new_res1_type)

		// add
		new_res1_quantity := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zjahcjum",
			"name": "res1_quantity",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_res1_quantity); err != nil {
			return err
		}
		collection.Schema.AddField(new_res1_quantity)

		// add
		new_res1_capacity := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "bgka4ang",
			"name": "res1_capacity",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_res1_capacity); err != nil {
			return err
		}
		collection.Schema.AddField(new_res1_capacity)

		// add
		new_res2_type := &schema.SchemaField{}
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
		}`), new_res2_type); err != nil {
			return err
		}
		collection.Schema.AddField(new_res2_type)

		// add
		new_res2_quantity := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ikggoskz",
			"name": "res2_quantity",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_res2_quantity); err != nil {
			return err
		}
		collection.Schema.AddField(new_res2_quantity)

		// add
		new_res2_capacity := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "mauixs2c",
			"name": "res2_capacity",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_res2_capacity); err != nil {
			return err
		}
		collection.Schema.AddField(new_res2_capacity)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cz4eps74xi3j7m7")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("cgbioj6d")

		// remove
		collection.Schema.RemoveField("zjahcjum")

		// remove
		collection.Schema.RemoveField("bgka4ang")

		// remove
		collection.Schema.RemoveField("bu2f4vpf")

		// remove
		collection.Schema.RemoveField("ikggoskz")

		// remove
		collection.Schema.RemoveField("mauixs2c")

		return dao.SaveCollection(collection)
	})
}
