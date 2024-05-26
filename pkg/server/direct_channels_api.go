package server

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

func createDirectChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,lowercase,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		Members     []uint `json:"members"`
		IsEncrypted bool   `json:"is_encrypted"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	} else if err = services.GetChannelAliasAvailability(data.Alias); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var realm *models.Realm
	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ExternalID, user.ExternalID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can create channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can create channel related to it")
		} else {
			realm = &val
		}
	}

	var members []models.Account
	if err := database.C.Where("id IN ?", data.Members).Find(&members).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	channel := models.Channel{
		Alias:       data.Alias,
		Name:        data.Name,
		Description: data.Description,
		IsEncrypted: data.IsEncrypted,
		AccountID:   user.ID,
		Type:        models.ChannelTypeDirect,
		Members: append([]models.ChannelMember{
			{AccountID: user.ID, PowerLevel: 100},
		}, lo.Map(members, func(item models.Account, idx int) models.ChannelMember {
			return models.ChannelMember{AccountID: item.ID, PowerLevel: 100}
		})...),
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
