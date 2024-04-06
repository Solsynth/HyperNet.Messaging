package database

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"gorm.io/gorm"
)

var DatabaseAutoActionRange = []any{
	&models.Account{},
	&models.Channel{},
	&models.ChannelMember{},
	&models.Call{},
	&models.Message{},
	&models.Attachment{},
}

func RunMigration(source *gorm.DB) error {
	if err := source.AutoMigrate(DatabaseAutoActionRange...); err != nil {
		return err
	}

	return nil
}
