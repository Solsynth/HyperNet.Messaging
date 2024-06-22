package services

import (
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"github.com/rs/zerolog/log"
)

func DoAutoDatabaseCleanup() {
	deadline := time.Now().Add(60 * time.Minute)
	log.Debug().Time("deadline", deadline).Msg("Now cleaning up entire database...")

	// Deal soft-deletion
	var count int64
	for _, model := range database.DatabaseAutoActionRange {
		tx := database.C.Unscoped().Delete(model, "deleted_at >= ?", deadline)
		if tx.Error != nil {
			log.Error().Err(tx.Error).Msg("An error occurred when running database cleanup...")
		}
		count += tx.RowsAffected
	}

	log.Debug().Int64("affected", count).Msg("Clean up entire database accomplished.")
}
