package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `[
			{
				"id": "_pb_users_auth_",
				"created": "2025-06-01 05:39:51.574Z",
				"updated": "2025-06-01 07:53:34.251Z",
				"name": "users",
				"type": "auth",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "irv0icng",
						"name": "name",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "tzdey4t2",
						"name": "avatar",
						"type": "file",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"mimeTypes": [
								"image/jpeg",
								"image/png",
								"image/svg+xml",
								"image/gif",
								"image/webp"
							],
							"thumbs": null,
							"maxSelect": 1,
							"maxSize": 5242880,
							"protected": false
						}
					}
				],
				"indexes": [],
				"listRule": "id = @request.auth.id",
				"viewRule": "id = @request.auth.id",
				"createRule": "",
				"updateRule": "id = @request.auth.id",
				"deleteRule": "id = @request.auth.id",
				"options": {
					"allowEmailAuth": true,
					"allowOAuth2Auth": true,
					"allowUsernameAuth": true,
					"exceptEmailDomains": null,
					"manageRule": null,
					"minPasswordLength": 8,
					"onlyEmailDomains": null,
					"onlyVerified": false,
					"requireEmail": false
				}
			},
			{
				"id": "rf5ut7k9msu0y4u",
				"created": "2025-06-01 05:39:51.577Z",
				"updated": "2025-06-01 14:41:01.609Z",
				"name": "resource_types",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "k6xkhw8b",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": true,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "qb7mcgmk",
						"name": "description",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "302iyoob",
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
					},
					{
						"system": false,
						"id": "extudwdd",
						"name": "icon",
						"type": "url",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"exceptDomains": null,
							"onlyDomains": null
						}
					}
				],
				"indexes": [],
				"listRule": "",
				"viewRule": "",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "rsajrwzcgtu2ipb",
				"created": "2025-06-01 05:39:51.578Z",
				"updated": "2025-06-01 12:17:11.184Z",
				"name": "planet_types",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "phhkck3u",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": true,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "52rqv91v",
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
					},
					{
						"system": false,
						"id": "7ousf59p",
						"name": "icon",
						"type": "url",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"exceptDomains": null,
							"onlyDomains": null
						}
					}
				],
				"indexes": [],
				"listRule": "",
				"viewRule": "",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "f05gg2gk3nwq94i",
				"created": "2025-06-01 05:39:51.578Z",
				"updated": "2025-06-03 01:20:12.284Z",
				"name": "building_types",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "svbi8rsn",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": true,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "aerontfy",
						"name": "cost",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "e0tnqunx",
						"name": "strength",
						"type": "select",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"strong",
								"weak",
								"na"
							]
						}
					},
					{
						"system": false,
						"id": "jbzjxnkm",
						"name": "power_consumption",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "fpjw0xiy",
						"name": "res1_type",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "3lfzqjhn",
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
					},
					{
						"system": false,
						"id": "8d5cmjbr",
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
					},
					{
						"system": false,
						"id": "3rs7tpyw",
						"name": "res2_type",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "1fqxoazi",
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
					},
					{
						"system": false,
						"id": "u44nu6rd",
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
					},
					{
						"system": false,
						"id": "twxcvap5",
						"name": "description",
						"type": "editor",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"convertUrls": false
						}
					},
					{
						"system": false,
						"id": "fqkoehw1",
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
					},
					{
						"system": false,
						"id": "nqxcmwwc",
						"name": "cost_resource_type",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "b4btc1zg",
						"name": "cost_quantity",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "hdb75tx5bgaru5z",
				"created": "2025-06-01 05:39:51.578Z",
				"updated": "2025-06-01 05:39:51.578Z",
				"name": "ship_types",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "gfm6itwz",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": true,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "62hxp7ov",
						"name": "cost",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "ky3bnxj6",
						"name": "strength",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "pc5nvltu",
						"name": "cargo_capacity",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "37qvu786zuy0bpn",
				"created": "2025-06-01 05:39:51.579Z",
				"updated": "2025-06-01 05:39:51.579Z",
				"name": "systems",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "j46vjjxb",
						"name": "name",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "wgcry5ph",
						"name": "x",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "qvp2ckev",
						"name": "y",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "abvtmqyn",
						"name": "discovered_by",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX idx_systems_xy ON systems (x, y)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "gki7vxfevkkctvo",
				"created": "2025-06-01 05:39:51.579Z",
				"updated": "2025-06-01 05:39:51.579Z",
				"name": "planets",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "ecp1vtzg",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "f7lpq4ki",
						"name": "system_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "systems",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "eomlhs60",
						"name": "planet_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "planet_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "snze1bbu",
						"name": "size",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "oc4op8ci",
						"name": "colonized_by",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "wbancteh",
						"name": "colonized_at",
						"type": "date",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					}
				],
				"indexes": [
					"CREATE INDEX idx_planets_system ON planets (system_id)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": "@request.auth.id != ''",
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "qi5msf628j84qo2",
				"created": "2025-06-01 05:39:51.580Z",
				"updated": "2025-06-01 15:01:32.274Z",
				"name": "resource_nodes",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "kpji2igz",
						"name": "planet_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "planets",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "rydpv0yd",
						"name": "resource_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "ibwvm0qg",
						"name": "richness",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "evuzhfcr",
						"name": "exhausted",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					}
				],
				"indexes": [
					"CREATE INDEX idx_resource_nodes_planet ON resource_nodes (planet_id)"
				],
				"listRule": "",
				"viewRule": "",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "ccwzdrc7c4vgmma",
				"created": "2025-06-01 05:39:51.580Z",
				"updated": "2025-06-02 23:46:40.067Z",
				"name": "fleets",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "9ke6poqm",
						"name": "owner_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": 1,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "r079algi",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "iuqi3ut4",
						"name": "current_system",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "37qvu786zuy0bpn",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_fleets_owner` + "`" + ` ON ` + "`" + `fleets` + "`" + ` (` + "`" + `owner_id` + "`" + `)"
				],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"options": {}
			},
			{
				"id": "81uj97m140vekd7",
				"created": "2025-06-01 05:39:51.580Z",
				"updated": "2025-06-01 05:39:51.580Z",
				"name": "trade_routes",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "tunhosaf",
						"name": "owner_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "mmd9xyyi",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "vukzpghq",
						"name": "from_system",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "systems",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "uzzjgdvc",
						"name": "to_system",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "systems",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "n0loj800",
						"name": "resource_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "vtfwrue1",
						"name": "active",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"options": {}
			},
			{
				"id": "8efqt6p5vuwvhf8",
				"created": "2025-06-01 05:39:51.581Z",
				"updated": "2025-06-01 05:39:51.581Z",
				"name": "ships",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "fmkbjkin",
						"name": "fleet_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fleets",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "saicvpda",
						"name": "ship_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "ship_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "lvrmwnyy",
						"name": "count",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "vozutvyv",
						"name": "health",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [
					"CREATE INDEX idx_ships_fleet ON ships (fleet_id)"
				],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
				"options": {}
			},
			{
				"id": "ejw6or896geb6e4",
				"created": "2025-06-01 05:39:51.581Z",
				"updated": "2025-06-01 05:39:51.581Z",
				"name": "populations",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "q7cwndlh",
						"name": "owner_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "choajjzu",
						"name": "planet_id",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "planets",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "byuzwegk",
						"name": "fleet_id",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fleets",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "pvq0eny8",
						"name": "employed_at",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "buildings",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "fv2gqiun",
						"name": "count",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "khkmeeow",
						"name": "happiness",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [
					"CREATE INDEX idx_populations_owner ON populations (owner_id)"
				],
				"listRule": "owner_id = @request.auth.id",
				"viewRule": "owner_id = @request.auth.id",
				"createRule": "@request.auth.id != '' && @request.data.owner_id = @request.auth.id",
				"updateRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"deleteRule": "@request.auth.id != '' && owner_id = @request.auth.id",
				"options": {}
			},
			{
				"id": "3rlbvt8e4ggy51j",
				"created": "2025-06-01 05:39:51.581Z",
				"updated": "2025-06-01 07:38:26.253Z",
				"name": "buildings",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "06d9gojl",
						"name": "planet_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "gki7vxfevkkctvo",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "qzpedlip",
						"name": "building_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "f05gg2gk3nwq94i",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "rnvguxw2",
						"name": "level",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "v9oypgpy",
						"name": "active",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "fn8tbg54",
						"name": "completion_time",
						"type": "date",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
						}
					},
					{
						"system": false,
						"id": "iuffn9ji",
						"name": "resource_nodes",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "qi5msf628j84qo2",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 5,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "ipp9vvkj",
						"name": "res1_stored",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": true
						}
					},
					{
						"system": false,
						"id": "awsrbvdm",
						"name": "res2_stored",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": true
						}
					}
				],
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_buildings_planet` + "`" + ` ON ` + "`" + `buildings` + "`" + ` (` + "`" + `planet_id` + "`" + `)"
				],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
				"options": {}
			},
			{
				"id": "47dvousve2bgt4z",
				"created": "2025-06-01 21:05:26.033Z",
				"updated": "2025-06-01 21:05:26.033Z",
				"name": "hyperlanes",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "rsotja45",
						"name": "from_system",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "systems",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "kynrbcmi",
						"name": "to_system",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "systems",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "3bp6yegm",
						"name": "distance",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [],
				"listRule": "",
				"viewRule": "",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "5r6mvf4afhfwsku",
				"created": "2025-06-02 22:45:19.845Z",
				"updated": "2025-06-03 00:12:19.085Z",
				"name": "fleet_orders",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "u0egorse",
						"name": "user_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "_pb_users_auth_",
							"cascadeDelete": false,
							"minSelect": 1,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "znndtx6w",
						"name": "fleet_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "ccwzdrc7c4vgmma",
							"cascadeDelete": true,
							"minSelect": 1,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "n1lt1dwl",
						"name": "type",
						"type": "select",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"move"
							]
						}
					},
					{
						"system": false,
						"id": "z1ptjsy5",
						"name": "status",
						"type": "select",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"pending",
								"processing",
								"completed",
								"failed",
								"cancelled"
							]
						}
					},
					{
						"system": false,
						"id": "ip9gzn7n",
						"name": "execute_at_tick",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "jkjjqc2f",
						"name": "destination_system_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "37qvu786zuy0bpn",
							"cascadeDelete": false,
							"minSelect": 1,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "yru67szq",
						"name": "original_system_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "37qvu786zuy0bpn",
							"cascadeDelete": false,
							"minSelect": 1,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "ghedbz8b",
						"name": "travel_time_ticks",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 1,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "kkds59nv",
						"name": "route_path",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 10240
						}
					},
					{
						"system": false,
						"id": "pep4cqpz",
						"name": "current_hop",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "6pt7xjcn",
						"name": "final_destination_id",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "37qvu786zuy0bpn",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_fleet_orders_user_status` + "`" + ` ON ` + "`" + `fleet_orders` + "`" + ` (\n  ` + "`" + `user_id` + "`" + `,\n  ` + "`" + `status` + "`" + `\n)",
					"CREATE INDEX ` + "`" + `idx_fleet_orders_status_execute_at_tick` + "`" + ` ON ` + "`" + `fleet_orders` + "`" + ` (\n  ` + "`" + `status` + "`" + `,\n  ` + "`" + `execute_at_tick` + "`" + `\n)",
					"CREATE INDEX ` + "`" + `idx_fleet_orders_fleet_status` + "`" + ` ON ` + "`" + `fleet_orders` + "`" + ` (\n  ` + "`" + `fleet_id` + "`" + `,\n  ` + "`" + `status` + "`" + `\n)"
				],
				"listRule": "@request.auth.id = user_id",
				"viewRule": "@request.auth.id = user_id",
				"createRule": "@request.auth.id != \"\"",
				"updateRule": "@request.auth.id = user_id && @request.data.status = \"cancelled\" && status = \"pending\"",
				"deleteRule": "@request.auth.id = user_id && (status = \"failed\" || status = \"cancelled\")",
				"options": {}
			},
			{
				"id": "ship_cargo",
				"created": "2025-06-03 01:15:00.000Z",
				"updated": "2025-06-03 01:15:00.000Z",
				"name": "ship_cargo",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "ship_id",
						"name": "ship_id",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "ships",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "resource_type",
						"name": "resource_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "resource_types",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "quantity",
						"name": "quantity",
						"type": "number",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": null,
							"noDecimal": false
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX idx_ship_cargo_unique ON ship_cargo (ship_id, resource_type)",
					"CREATE INDEX idx_ship_cargo_ship ON ship_cargo (ship_id)"
				],
				"listRule": "@request.auth.id != ''",
				"viewRule": "@request.auth.id != ''",
				"createRule": "@request.auth.id != ''",
				"updateRule": "@request.auth.id != ''",
				"deleteRule": "@request.auth.id != ''",
				"options": {}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}
