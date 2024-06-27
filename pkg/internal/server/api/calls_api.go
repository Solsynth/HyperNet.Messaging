package api

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
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
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
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
		_, _ = services.NewEvent(models.Event{
			Uuid:      uuid.NewString(),
			Body:      map[string]any{},
			Type:      "calls.start",
			Channel:   channel,
			Sender:    membership,
			ChannelID: channel.ID,
			SenderID:  membership.ID,
		})

		return c.JSON(call)
	}
}

func endCall(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
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
		_, _ = services.NewEvent(models.Event{
			Uuid:      uuid.NewString(),
			Body:      map[string]any{"last": call.EndedAt.Unix() - call.CreatedAt.Unix()},
			Type:      "calls.end",
			Channel:   channel,
			Sender:    membership,
			ChannelID: channel.ID,
			SenderID:  membership.ID,
		})

		return c.JSON(call)
	}
}

func exchangeCallToken(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)
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

	tk, err := services.EncodeCallToken(user, call)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(fiber.Map{
			"token":    tk,
			"endpoint": viper.GetString("calling.endpoint"),
		})
	}
}
