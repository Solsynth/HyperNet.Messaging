package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/http/exts"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func createDirectChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,lowercase,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		RelatedUser uint   `json:"related_user"`
		IsEncrypted bool   `json:"is_encrypted"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else if err = services.GetChannelAliasAvailability(data.Alias); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var realm *models.Realm
	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ID, user.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can create channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can create channel related to it")
		} else {
			realm = &val
		}
	}

	var relatedUser authm.Account
	if err := database.C.
		Where("external_id = ?", data.RelatedUser).
		First(&relatedUser).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find related user: %v", err))
	}

	if ch, err := services.GetDirectChannelByUser(user, relatedUser); err == nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("you already have a direct with that user #%d", ch.ID))
	}

	channel := models.Channel{
		Alias:       data.Alias,
		Name:        data.Name,
		Description: data.Description,
		IsPublic:    false,
		IsCommunity: false,
		AccountID:   user.ID,
		Type:        models.ChannelTypeDirect,
		Members: []models.ChannelMember{
			{AccountID: user.ID, PowerLevel: 100},
			{AccountID: relatedUser.ID, PowerLevel: 100},
		},
	}

	if realm != nil {
		channel.RealmID = &realm.ID
	}

	channel, err := services.NewChannel(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}
