package services

import (
	"context"
	"fmt"
	localCache "git.solsynth.dev/hypernet/messaging/pkg/internal/cache"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
)

func ListChannelMember(channelId uint) ([]models.ChannelMember, error) {
	var members []models.ChannelMember

	if err := database.C.
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
		Find(&member).Error; err != nil {
		return member, err
	}

	return member, nil
}

func AddChannelMemberWithCheck(user authm.Account, target models.Channel) error {
	if err := authkit.EnsureUserPermGranted(gap.Nx, user.ID, target.AccountID, "ChannelAdd", true); err != nil {
		return fmt.Errorf("unable to add user into your channel due to access denied: %v", err)
	}

	member := models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error
	return err
}

func AddChannelMember(user authm.Account, target models.Channel) error {
	member := models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}

	err := database.C.Save(&member).Error

	if err == nil {
		cacheManager := cache.New[any](localCache.S)
		marshal := marshaler.New(cacheManager)
		contx := context.Background()

		_ = marshal.Invalidate(
			contx,
			store.WithInvalidateTags([]string{
				fmt.Sprintf("channel#%d", target.ID),
				fmt.Sprintf("user#%d", user.ID),
			}),
		)
	}

	return err
}

func EditChannelMember(membership models.ChannelMember) (models.ChannelMember, error) {
	if err := database.C.Save(&membership).Error; err != nil {
		return membership, err
	} else {
		cacheManager := cache.New[any](localCache.S)
		marshal := marshaler.New(cacheManager)
		contx := context.Background()

		_ = marshal.Invalidate(
			contx,
			store.WithInvalidateTags([]string{
				fmt.Sprintf("channel#%d", membership.ChannelID),
				fmt.Sprintf("user#%d", membership.AccountID),
			}),
		)
	}

	return membership, nil
}

func RemoveChannelMember(user authm.Account, target models.Channel) error {
	var member models.ChannelMember

	if err := database.C.Where(&models.ChannelMember{
		ChannelID: target.ID,
		AccountID: user.ID,
	}).First(&member).Error; err != nil {
		return err
	}

	if err := database.C.Delete(&member).Error; err == nil {
		database.C.Where("sender_id = ?").Delete(&models.Event{})

		cacheManager := cache.New[any](localCache.S)
		marshal := marshaler.New(cacheManager)
		contx := context.Background()

		_ = marshal.Invalidate(
			contx,
			store.WithInvalidateTags([]string{
				fmt.Sprintf("channel#%d", target.ID),
				fmt.Sprintf("user#%d", user.ID),
			}),
		)

		return nil
	} else {
		return err
	}
}
