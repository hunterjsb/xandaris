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

		collection, err := dao.FindCollectionByNameOrId("banks")
		if err != nil {
			return err
		}

		// Update owner_id field to be text instead of relation
		for i, field := range collection.Schema.Fields() {
			if field.Name == "owner_id" {
				newField := &schema.SchemaField{}
				json.Unmarshal([]byte(`{
					"system": false,
					"id": "owner_id",
					"name": "owner_id",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				}`), newField)
				collection.Schema.Fields()[i] = newField
			}
			if field.Name == "system_id" {
				newField := &schema.SchemaField{}
				json.Unmarshal([]byte(`{
					"system": false,
					"id": "system_id",
					"name": "system_id",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				}`), newField)
				collection.Schema.Fields()[i] = newField
			}
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("banks")
		if err != nil {
			return err
		}

		// Revert owner_id field to be relation
		for i, field := range collection.Schema.Fields() {
			if field.Name == "owner_id" {
				newField := &schema.SchemaField{}
				json.Unmarshal([]byte(`{
					"system": false,
					"id": "owner_id",
					"name": "owner_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": null,
						"displayFields": null
					}
				}`), newField)
				collection.Schema.Fields()[i] = newField
			}
			if field.Name == "system_id" {
				newField := &schema.SchemaField{}
				json.Unmarshal([]byte(`{
					"system": false,
					"id": "system_id",
					"name": "system_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "systems",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": null,
						"displayFields": null
					}
				}`), newField)
				collection.Schema.Fields()[i] = newField
			}
		}

		return dao.SaveCollection(collection)
	})
}