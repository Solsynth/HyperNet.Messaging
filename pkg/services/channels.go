package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/samber/lo"
)

func ListChannel() ([]models.Channel, error) {
	var channels []models.Channel
	if err := database.C.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListChannelWithUser(user models.Account) ([]models.Channel, error) {
	var channels []models.Channel
	if err := database.C.Where(&models.Channel{AccountID: user.ID}).Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListChannelIsAvailable(user models.Account) ([]models.Channel, error) {
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

	if err := database.C.Where("id IN ?", idx).Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func NewChannel(user models.Account, name, description string) (models.Channel, error) {
	channel := models.Channel{
		Name:        name,
		Description: description,
		AccountID:   user.ID,
		Members: []models.ChannelMember{
			{AccountID: user.ID},
		},
	}

	err := database.C.Save(&channel).Error

	return channel, err
}

func ListChannelMember(channelId uint) ([]models.ChannelMember, error) {
	var members []models.ChannelMember

	if err := database.C.
		Where(&models.ChannelMember{ChannelID: channelId}).
		Preload("Account").
		Find(&members).Error; err != nil {
		return members, err
	}

	return members, nil
}

func InviteChannelMember(user models.Account, target models.Channel) error {
	member := models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error

	return err
}

func KickChannelMember(user models.Account, target models.Channel) error {
	var member models.ChannelMember

	if err := database.C.Where(&models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}).First(&member).Error; err != nil {
		return err
	}

	return database.C.Delete(&member).Error
}

func EditChannel(channel models.Channel, name, description string) (models.Channel, error) {
	channel.Name = name
	channel.Description = description

	err := database.C.Save(&channel).Error

	return channel, err
}

func DeleteChannel(channel models.Channel) error {
	return database.C.Delete(&channel).Error
}
