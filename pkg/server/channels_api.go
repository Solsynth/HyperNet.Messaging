package server

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
)

func getChannel(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("channelId", 0)

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		BaseModel: models.BaseModel{ID: uint(id)},
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(channel)
}

func listChannel(c *fiber.Ctx) error {
	channels, err := services.ListChannel()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listOwnedChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)

	channels, err := services.ListChannelWithUser(user)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listAvailableChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)

	channels, err := services.ListChannelIsAvailable(user)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func createChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	channel, err := services.NewChannel(user, data.Alias, data.Name, data.Description)
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

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		BaseModel: models.BaseModel{ID: uint(id)},
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
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

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		BaseModel: models.BaseModel{ID: uint(id)},
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.DeleteChannel(channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
