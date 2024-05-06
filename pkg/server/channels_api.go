package server

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
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

func getChannelAvailability(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channel, _, err = services.GetAvailableChannelWithAlias(alias, user, val.ID)
	} else {
		channel, _, err = services.GetAvailableChannelWithAlias(alias, user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	return c.JSON(channel)
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
	user := c.Locals("principal").(models.Account)

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
	user := c.Locals("principal").(models.Account)

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
	user := c.Locals("principal").(models.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,lowercase,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	} else if err = services.GetChannelAliasAvailability(data.Alias); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var realm *models.Realm
	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ExternalID, user.ExternalID); err != nil {
			return fmt.Errorf("you must be a part of that realm then can create channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fmt.Errorf("you must be a moderator of that realm then can create channel related to it")
		} else {
			realm = &val
		}
	}

	var err error
	var channel models.Channel
	if realm != nil {
		channel, err = services.NewChannel(user, data.Alias, data.Name, data.Description, realm.ID)
	} else {
		channel, err = services.NewChannel(user, data.Alias, data.Name, data.Description)
	}

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}

func editChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	id, _ := c.ParamsInt("channelId", 0)

	var data struct {
		Alias       string `json:"alias" validate:"required,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	tx := database.C.Where(&models.Channel{BaseModel: models.BaseModel{ID: uint(id)}})

	if val, ok := c.Locals("realm").(models.Realm); ok {
		if info, err := services.GetRealmMember(val.ExternalID, user.ExternalID); err != nil {
			return fmt.Errorf("you must be a part of that realm then can edit channel related to it")
		} else if info.GetPowerLevel() < 50 {
			return fmt.Errorf("you must be a moderator of that realm then can edit channel related to it")
		} else {
			tx = tx.Where("realm_id = ?", val.ID)
		}
	} else {
		tx = tx.Where("account_id = ? AND realm_id IS NULL", user.ID)
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	channel, err := services.EditChannel(channel, data.Alias, data.Name, data.Description)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}

func deleteChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
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
		tx = tx.Where("account_id = ? AND realm_id IS NULL", user.ID)
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.DeleteChannel(channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
