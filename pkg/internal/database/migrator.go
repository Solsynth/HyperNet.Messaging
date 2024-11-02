package database

import (
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"gorm.io/gorm"
)

var AutoMaintainRange = []any{
	&authm.Realm{},
	&models.Channel{},
	&models.ChannelMember{},
	&models.Call{},
	&models.Event{},
}

func RunMigration(source *gorm.DB) error {
	if err := source.AutoMigrate(AutoMaintainRange...); err != nil {
		return err
	}

	return nil
}
