package services

import (
	"fmt"
	"regexp"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func GetChannelAliasAvailability(alias string) error {
	if !regexp.MustCompile("^[a-z0-9-]+$").MatchString(alias) {
		return fmt.Errorf("channel alias should only contains lowercase letters, numbers, and hyphens")
	}
	return nil
}

func GetChannel(id uint) (models.Channel, error) {
	var channel models.Channel
	tx := database.C.Where(models.Channel{
		BaseModel: hyper.BaseModel{ID: id},
	}).Preload("Account").Preload("Realm")
	tx = PreloadDirectChannelMembers(tx)
	if err := tx.First(&channel).Error; err != nil {
		return channel, err
	}

	return channel, nil
}

func GetChannelWithAlias(alias string, realmId ...uint) (models.Channel, error) {
	var channel models.Channel
	tx := database.C.Where(models.Channel{Alias: alias}).Preload("Account").Preload("Realm")
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	} else {
		tx = tx.Where("realm_id IS NULL")
	}
	tx = PreloadDirectChannelMembers(tx)
	if err := tx.First(&channel).Error; err != nil {
		return channel, err
	}

	return channel, nil
}

func GetAvailableChannelWithAlias(alias string, user models.Account, realmId ...uint) (models.Channel, models.ChannelMember, error) {
	var err error
	var member models.ChannelMember
	var channel models.Channel
	if channel, err = GetChannelWithAlias(alias, realmId...); err != nil {
		return channel, member, err
	}

	if err := database.C.Where(models.ChannelMember{
		AccountID: user.ID,
		ChannelID: channel.ID,
	}).First(&member).Error; err != nil {
		return channel, member, fmt.Errorf("channel principal not found: %v", err.Error())
	}

	return channel, member, nil
}

func GetAvailableChannel(id uint, user models.Account) (models.Channel, models.ChannelMember, error) {
	var err error
	var member models.ChannelMember
	var channel models.Channel
	if channel, err = GetChannel(id); err != nil {
		return channel, member, err
	}
	tx := database.C.Where(models.ChannelMember{
		AccountID: user.ID,
		ChannelID: channel.ID,
	})
	if err := tx.First(&member).Error; err != nil {
		return channel, member, fmt.Errorf("channel principal not found: %v", err.Error())
	}

	return channel, member, nil
}

func PreloadDirectChannelMembers(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Members", func(db *gorm.DB) *gorm.DB {
		return db.Joins(
			fmt.Sprintf(
				"JOIN %schannels AS c ON c.type = ?",
				viper.GetString("database.prefix"),
			),
			models.ChannelTypeDirect,
		)
	}).Preload("Members.Account")
}

func ListChannel(user *models.Account, realmId ...uint) ([]models.Channel, error) {
	var identities []models.ChannelMember
	var idRange []uint
	if user != nil {
		if err := database.C.Where("account_id = ?", user.ID).Find(&identities).Error; err != nil {
			return nil, fmt.Errorf("unabkle to get identities: %v", err)
		}
		for _, identity := range identities {
			idRange = append(idRange, identity.ChannelID)
		}
	}

	var channels []models.Channel
	tx := database.C.Preload("Account").Preload("Realm")
	tx = tx.Where("id IN ? OR is_public = true", idRange)
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListChannelWithUser(user models.Account, realmId ...uint) ([]models.Channel, error) {
	var channels []models.Channel
	tx := database.C.Where(&models.Channel{AccountID: user.ID}).Preload("Realm")
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListAvailableChannel(user models.Account, realmId ...uint) ([]models.Channel, error) {
	var channels []models.Channel
	var members []models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		AccountID: user.ID,
	}).Find(&members).Error; err != nil {
		return channels, err
	}

	idx := lo.Map(members, func(item models.ChannelMember, index int) uint {
		return item.ChannelID
	})

	tx := database.C.Preload("Realm").Where("id IN ?", idx)
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func NewChannel(channel models.Channel) (models.Channel, error) {
	err := database.C.Save(&channel).Error
	return channel, err
}

func EditChannel(channel models.Channel, alias, name, description string, isPublic, isCommunity bool) (models.Channel, error) {
	channel.Alias = alias
	channel.Name = name
	channel.Description = description
	channel.IsPublic = isPublic
	channel.IsCommunity = isCommunity

	err := database.C.Save(&channel).Error

	return channel, err
}

func DeleteChannel(channel models.Channel) error {
	return database.C.Delete(&channel).Error
}
