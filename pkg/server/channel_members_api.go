package server

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
)

func listChannelMembers(c *fiber.Ctx) error {
	channelId, _ := c.ParamsInt("channelId", 0)

	if members, err := services.ListChannelMember(uint(channelId)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(members)
	}
}

func inviteChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	channelId, _ := c.ParamsInt("channelId", 0)

	var data struct {
		AccountName string `json:"account_name" validate:"required"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		BaseModel: models.BaseModel{ID: uint(channelId)},
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var account models.Account
	if err := database.C.Where(&models.Account{
		Name: data.AccountName,
	}).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.InviteChannelMember(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func kickChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	channelId, _ := c.ParamsInt("channelId", 0)

	var data struct {
		AccountName string `json:"account_name" validate:"required"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		BaseModel: models.BaseModel{ID: uint(channelId)},
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var account models.Account
	if err := database.C.Where(&models.Account{
		Name: data.AccountName,
	}).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.KickChannelMember(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}
