package server

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"net/url"
)

func listCall(c *fiber.Ctx) error {
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if calls, err := services.ListCall(channel, take, offset); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else {
		return c.JSON(calls)
	}
}

func getOngoingCall(c *fiber.Ctx) error {
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if call, err := services.GetOngoingCall(channel); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else {
		return c.JSON(call)
	}
}

func startCall(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var membership models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channel.ID,
		AccountID: user.ID,
	}).Find(&membership).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	call, err := services.NewCall(channel, membership)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(call)
	}
}

func endCall(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var membership models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channel.ID,
		AccountID: user.ID,
	}).Find(&membership).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	call, err := services.GetOngoingCall(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if call.FounderID != user.ID && channel.AccountID != user.ID {
		return fiber.NewError(fiber.StatusBadRequest, "only call founder or channel owner can end this call")
	}

	if call, err := services.EndCall(call); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(call)
	}
}

func exchangeCallToken(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var membership models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channel.ID,
		AccountID: user.ID,
	}).Find(&membership).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	call, err := services.GetOngoingCall(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	tk, err := services.EncodeCallToken(user)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(fiber.Map{
			"token":    tk,
			"endpoint": viper.GetString("calling.endpoint"),
			"full_url": fmt.Sprintf(
				"%s/%s?jwt=%s",
				viper.GetString("calling.endpoint"),
				call.ExternalID,
				url.QueryEscape(tk),
			),
		})
	}
}
