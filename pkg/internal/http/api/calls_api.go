package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"sync"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/http/exts"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var callLocks sync.Map

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
	} else if res, err := services.GetCallParticipants(call); err != nil {
		return c.JSON(call)
	} else {
		call.Participants = res
		return c.JSON(call)
	}
}

func startCall(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "CreateCalls", true); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
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
	} else if membership.PowerLevel < 0 {
		return fiber.NewError(fiber.StatusForbidden, "you have not enough permission to create a call")
	}

	if _, ok := callLocks.Load(channel.ID); ok {
		return fiber.NewError(fiber.StatusLocked, "there is already a call in creation progress for this channel")
	} else {
		callLocks.Store(channel.ID, true)
	}

	call, err := services.NewCall(channel, membership)
	if err != nil {
		callLocks.Delete(channel.ID)
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

		callLocks.Delete(channel.ID)
		return c.JSON(call)
	}
}

func endCall(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
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
	} else if call.FounderID != membership.ID && membership.PowerLevel < 50 {
		return fiber.NewError(fiber.StatusBadRequest, "only call founder or channel moderator can end this call")
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

func kickParticipantInCall(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Username string `json:"username" validate:"required"`
	}
	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

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
	} else if call.FounderID != user.ID && membership.PowerLevel < 50 {
		return fiber.NewError(fiber.StatusBadRequest, "only call founder or channel admin can kick participant in this call")
	}

	if err = services.KickParticipantInCall(call, data.Username); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}

func exchangeCallToken(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
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
