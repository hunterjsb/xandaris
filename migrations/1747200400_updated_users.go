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
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		// add color field
		new_color := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "user_color",
			"name": "color",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": 7,
				"pattern": "^#[0-9A-Fa-f]{6}$"
			}
		}`), new_color); err != nil {
			return err
		}
		collection.Schema.AddField(new_color)

		// add alliance_id field
		new_alliance := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "user_alliance",
			"name": "alliance_id",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": 50,
				"pattern": ""
			}
		}`), new_alliance); err != nil {
			return err
		}
		collection.Schema.AddField(new_alliance)

		// add last_seen field
		new_last_seen := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "user_last_seen",
			"name": "last_seen",
			"type": "date",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": "",
				"max": ""
			}
		}`), new_last_seen); err != nil {
			return err
		}
		collection.Schema.AddField(new_last_seen)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		// remove fields
		collection.Schema.RemoveField("user_color")
		collection.Schema.RemoveField("user_alliance")
		collection.Schema.RemoveField("user_last_seen")

		return dao.SaveCollection(collection)
	})
}