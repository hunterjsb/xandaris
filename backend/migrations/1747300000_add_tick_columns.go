package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Add eta_tick column to fleets table
		fleetCollection, err := dao.FindCollectionByNameOrId("fleets")
		if err != nil {
			return err
		}

		// Add eta_tick field for tick-based timing
		zero := float64(0)
		fleetCollection.Schema.AddField(&schema.SchemaField{
			Name:     "eta_tick",
			Type:     schema.FieldTypeNumber,
			Required: false,
			Options: &schema.NumberOptions{
				Min: &zero,
			},
		})

		if err := dao.SaveCollection(fleetCollection); err != nil {
			return err
		}

		// Add eta_tick column to trade_routes table
		tradeCollection, err := dao.FindCollectionByNameOrId("trade_routes")
		if err != nil {
			return err
		}

		// Add eta_tick field for tick-based timing
		tradeCollection.Schema.AddField(&schema.SchemaField{
			Name:     "eta_tick",
			Type:     schema.FieldTypeNumber,
			Required: false,
			Options: &schema.NumberOptions{
				Min: &zero,
			},
		})

		if err := dao.SaveCollection(tradeCollection); err != nil {
			return err
		}

		return nil
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Rollback: Remove eta_tick columns
		fleetCollection, err := dao.FindCollectionByNameOrId("fleets")
		if err != nil {
			return err
		}

		fleetCollection.Schema.RemoveField("eta_tick")
		if err := dao.SaveCollection(fleetCollection); err != nil {
			return err
		}

		tradeCollection, err := dao.FindCollectionByNameOrId("trade_routes")
		if err != nil {
			return err
		}

		tradeCollection.Schema.RemoveField("eta_tick")
		if err := dao.SaveCollection(tradeCollection); err != nil {
			return err
		}

		return nil
	})
}