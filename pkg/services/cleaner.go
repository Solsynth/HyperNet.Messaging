package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/rs/zerolog/log"
	"time"
)

func DoAutoDatabaseCleanup() {
	deadline := time.Now().Add(60 * time.Minute)
	log.Debug().Time("deadline", deadline).Msg("Now cleaning up entire database...")

	// Deal soft-deletion
	var count int64
	for _, model := range database.DatabaseAutoActionRange {
		tx := database.C.Unscoped().Delete(model, "deleted_at >= ?", deadline)
		if tx.Error != nil {
			log.Error().Err(tx.Error).Msg("An error occurred when running auth context cleanup...")
		}
		count += tx.RowsAffected
	}

	// Clean up outdated chat history
	tx := database.C.Unscoped().Delete(&models.Message{}, "created_at < ?", time.Now().Add(30*24*time.Hour))
	if tx.Error != nil {
		log.Error().Err(tx.Error).Msg("An error occurred when running auth context cleanup...")
	} else {
		count += tx.RowsAffected
	}

	log.Debug().Int64("affected", count).Msg("Clean up entire database accomplished.")
}
