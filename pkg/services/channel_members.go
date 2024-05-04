package services

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
)

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

func AddChannelMemberWithCheck(user models.Account, target models.Channel) error {
	if _, err := GetAccountFriend(user.ID, target.AccountID, 1); err != nil {
		return fmt.Errorf("you only can invite your friends to your channel")
	}

	member := models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error
	return err
}

func AddChannelMember(user models.Account, target models.Channel) error {
	member := models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error
	return err
}

func EditChannelMember(membership models.ChannelMember) (models.ChannelMember, error) {
	if err := database.C.Save(&membership).Error; err != nil {
		return membership, err
	}
	return membership, nil
}

func RemoveChannelMember(user models.Account, target models.Channel) error {
	var member models.ChannelMember

	if err := database.C.Where(&models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}).First(&member).Error; err != nil {
		return err
	}

	return database.C.Delete(&member).Error
}
