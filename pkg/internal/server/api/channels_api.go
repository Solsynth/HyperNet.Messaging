package api

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/server/exts"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func getChannel(c *fiber.Ctx) error {
	alias := c.Params("channel")

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(channel)
}

func getChannelIdentity(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
	alias := c.Params("channel")

	var err error
	var member models.ChannelMember
	if val, ok := c.Locals("realm").(models.Realm); ok {
		_, member, err = services.GetAvailableChannelWithAlias(alias, user, val.ID)
	} else {
		_, member, err = services.GetAvailableChannelWithAlias(alias, user)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusForbidden)
	}

	return c.JSON(member)
}

func listChannel(c *fiber.Ctx) error {
	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channels, err = services.ListChannel(val.ID)
	} else {
		channels, err = services.ListChannel()
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listOwnedChannel(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channels, err = services.ListChannelWithUser(user, val.ID)
	} else {
		channels, err = services.ListChannelWithUser(user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listAvailableChannel(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channels, err = services.ListAvailableChannel(user, val.ID)
	} else {
		channels, err = services.ListAvailableChannel(user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func createChannel(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,lowercase,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		IsEncrypted bool   `json:"is_encrypted"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
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

	channel := models.Channel{
		Alias:       data.Alias,
		Name:        data.Name,
		Description: data.Description,
		IsEncrypted: data.IsEncrypted,
		AccountID:   user.ID,
		Type:        models.ChannelTypeCommon,
		Members: []models.ChannelMember{
			{AccountID: user.ID, PowerLevel: 100},
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

func editChannel(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
	id, _ := c.ParamsInt("channelId", 0)

	var data struct {
		Alias       string `json:"alias" validate:"required,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		IsEncrypted bool   `json:"is_encrypted"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	tx := database.C.Where(&models.Channel{BaseModel: models.BaseModel{ID: uint(id)}})

	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ExternalID, user.ExternalID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can edit channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can edit channel related to it")
		} else {
			tx = tx.Where("realm_id = ?", val.ID)
		}
	} else {
		tx = tx.Where("realm_id IS NULL")
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if channel.RealmID != nil {
		if member, err := services.GetChannelMember(user, channel.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of this channel to edit it")
		} else if member.PowerLevel < 100 {
			return fiber.NewError(fiber.StatusForbidden, "you must be channel admin to edit it")
		}
	}

	channel, err := services.EditChannel(channel, data.Alias, data.Name, data.Description, data.IsEncrypted)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}

func deleteChannel(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
	id, _ := c.ParamsInt("channelId", 0)

	tx := database.C.Where(&models.Channel{BaseModel: models.BaseModel{ID: uint(id)}})

	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ExternalID, user.ExternalID); err != nil {
			return fmt.Errorf("you must be a part of that realm then can delete channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fmt.Errorf("you must be a moderator of that realm then can delete channel related to it")
		} else {
			tx = tx.Where("realm_id = ?", val.ID)
		}
	} else {
		tx = tx.Where("(account_id = ? OR type = ?) AND realm_id IS NULL", user.ID, models.ChannelTypeDirect)
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if channel.Type == models.ChannelTypeDirect {
		if member, err := services.GetChannelMember(user, channel.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must related to this direct message if you want delete it")
		} else if member.PowerLevel < 100 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of this direct message if you want delete it")
		}
	}

	if err := services.DeleteChannel(channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
