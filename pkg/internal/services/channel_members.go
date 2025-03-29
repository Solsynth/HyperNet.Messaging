package services

import (
	"errors"
	"fmt"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cachekit"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"gorm.io/gorm"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
)

func CountChannelMember(channelId uint) (int64, error) {
	var count int64
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channelId,
	}).Model(&models.ChannelMember{}).Count(&count).Error; err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func ListChannelMember(channelId uint, take int, offset int) ([]models.ChannelMember, error) {
	var members []models.ChannelMember

	if err := database.C.
		Limit(take).Offset(offset).
		Where(&models.ChannelMember{ChannelID: channelId}).
		Find(&members).Error; err != nil {
		return members, err
	}

	return members, nil
}

func GetChannelMember(user authm.Account, channelId uint) (models.ChannelMember, error) {
	var member models.ChannelMember

	if err := database.C.
		Where(&models.ChannelMember{AccountID: user.ID, ChannelID: channelId}).
		First(&member).Error; err != nil {
		return member, err
	}

	return member, nil
}

func AddChannelMemberWithCheck(user, op authm.Account, target models.Channel) error {
	if user.ID != op.ID {
		if err := authkit.EnsureUserPermGranted(gap.Nx, user.ID, op.ID, "ChannelAdd", true); err != nil {
			return fmt.Errorf("unable to add user into your channel due to access denied: %v", err)
		}
	}

	return AddChannelMember(user, target)
}

func AddChannelMember(user authm.Account, target models.Channel) error {
	var member models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		AccountID: user.ID,
		ChannelID: target.ID,
	}).First(&member).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	member = models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error

	if err == nil {
		cachekit.DeleteByTags(
			gap.Ca,
			fmt.Sprintf("channel#%d", target.ID),
			fmt.Sprintf("user#%d", user.ID),
		)
	}

	return err
}

func EditChannelMember(membership models.ChannelMember) (models.ChannelMember, error) {
	if err := database.C.Save(&membership).Error; err != nil {
		return membership, err
	} else {
		cachekit.DeleteByTags(
			gap.Ca,
			fmt.Sprintf("channel#%d", membership.ChannelID),
			fmt.Sprintf("user#%d", membership.AccountID),
		)
	}

	return membership, nil
}

func RemoveChannelMember(member models.ChannelMember, target models.Channel) error {
	if err := database.C.Delete(&member).Error; err == nil {
		database.C.Where("sender_id = ?").Delete(&models.Event{})

		cachekit.DeleteByTags(
			gap.Ca,
			fmt.Sprintf("channel#%d", target.ID),
			fmt.Sprintf("user#%d", member.AccountID),
		)

		return nil
	} else {
		return err
	}
}
