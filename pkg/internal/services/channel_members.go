package services

import (
	"fmt"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
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

func GetChannelMember(user models.Account, channelId uint) (models.ChannelMember, error) {
	var member models.ChannelMember

	if err := database.C.
		Where(&models.ChannelMember{AccountID: user.ID, ChannelID: channelId}).
		Find(&member).Error; err != nil {
		return member, err
	}

	return member, nil
}

func AddChannelMemberWithCheck(user models.Account, target models.Channel) error {
	if err := CheckUserPerm(user.ID, target.AccountID, "ChannelAdd", true); err != nil {
		return fmt.Errorf("unable to add user into your channel")
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
