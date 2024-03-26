package database

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"gorm.io/gorm"
)

func RunMigration(source *gorm.DB) error {
	if err := source.AutoMigrate(
		&models.Account{},
		&models.Channel{},
		&models.ChannelMember{},
		&models.Attachment{},
	); err != nil {
		return err
	}

	return nil
}
