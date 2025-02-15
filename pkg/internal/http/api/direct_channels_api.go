package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/http/exts"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
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
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else if err = services.GetChannelAliasAvailability(data.Alias); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var realm *authm.Realm
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		if info, err := authkit.GetRealmMember(gap.Nx, val.ID, user.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can create channel related to it")
		} else if info.PowerLevel < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can create channel related to it")
		} else {
			realm = &val
		}
	}

	relatedUser, err := authkit.GetUser(gap.Nx, data.RelatedUser)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find related user: %v", err))
	}

	if ch, err := services.GetDirectChannelByUser(user, relatedUser); err == nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("you already have a direct with that user #%d", ch.ID))
	}

	if err := authkit.EnsureUserPermGranted(gap.Nx, user.ID, relatedUser.ID, "ChannelAdd", true); err != nil {
		return fmt.Errorf("unable to add user into your channel due to access denied: %v", err)
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

	channel, err = services.NewChannel(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}
