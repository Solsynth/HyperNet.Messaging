package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"github.com/samber/lo"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/web/exts"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func getEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")
	id, _ := c.ParamsInt("eventId")

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("you need join the channel before you read the messages: %v", err))
	}

	event, err := services.GetEvent(channel.ID, uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(event)
}

func listEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)
	alias := c.Params("channel")

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("you need join the channel before you read the messages: %v", err))
	}

	count := services.CountEvent(channel)
	events, err := services.ListEvent(channel, take, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  events,
	})
}

func checkHasNewEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	pivot := c.QueryInt("pivot", 0)
	alias := c.Params("channel")

	if pivot < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "pivot must be greater than zero")
	}

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("you need join the channel before you read the messages: %v", err))
	}

	var count int64
	if err = database.C.
		Where("channel_id = ?", channel.ID).
		Where("id > ?", pivot).
		Model(&models.Event{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(fiber.Map{
			"up_to_date": lo.Ternary(count > 0, false, true),
			"count":      count,
		})
	}
}

func newRawEvent(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "CreateMessagingRawEvent", true); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Uuid string         `json:"uuid" validate:"required"`
		Type string         `json:"type" validate:"required"`
		Body map[string]any `json:"body"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else if len(data.Uuid) < 36 {
		return fiber.NewError(fiber.StatusBadRequest, "message uuid was not valid")
	}

	var err error
	var channel models.Channel
	var member models.ChannelMember

	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, member, err = services.GetChannelIdentity(alias, user.ID, val)
	} else {
		channel, member, err = services.GetChannelIdentity(alias, user.ID)
	}

	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if member.PowerLevel < 0 {
		return fiber.NewError(fiber.StatusForbidden, "you have not enough permission to send message")
	}

	event := models.Event{
		Uuid:      data.Uuid,
		Body:      data.Body,
		Type:      data.Type,
		Sender:    member,
		Channel:   channel,
		ChannelID: channel.ID,
		SenderID:  member.ID,
	}

	if event, err = services.NewEvent(event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(event)
}
