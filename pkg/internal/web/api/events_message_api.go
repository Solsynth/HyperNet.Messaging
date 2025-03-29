package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"strings"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/web/exts"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

func newMessageEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Uuid string                  `json:"uuid" validate:"required"`
		Type string                  `json:"type" validate:"required"`
		Body models.EventMessageBody `json:"body"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else if len(data.Uuid) < 36 {
		return fiber.NewError(fiber.StatusBadRequest, "message uuid was not valid")
	}

	data.Body.Text = strings.TrimSpace(data.Body.Text)
	if len(data.Body.Text) == 0 && len(data.Body.Attachments) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty message was not allowed")
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
		return fiber.NewError(fiber.StatusForbidden, "unable to send message, access denied")
	}

	var parsed map[string]any
	raw, _ := jsoniter.Marshal(data.Body)
	_ = jsoniter.Unmarshal(raw, &parsed)

	event := models.Event{
		Uuid:           data.Uuid,
		Body:           parsed,
		Type:           data.Type,
		Sender:         member,
		Channel:        channel,
		QuoteEventID:   data.Body.QuoteEventID,
		RelatedEventID: data.Body.RelatedEventID,
		ChannelID:      channel.ID,
		SenderID:       member.ID,
	}

	if event, err = services.NewEvent(event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(event)
}

func editMessageEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")
	messageId, _ := c.ParamsInt("messageId", 0)

	var data struct {
		Uuid string                  `json:"uuid" validate:"required"`
		Type string                  `json:"type" validate:"required"`
		Body models.EventMessageBody `json:"body"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	if len(data.Body.Text) == 0 && len(data.Body.Attachments) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "you cannot send an empty message")
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
	}

	var event models.Event
	if event, err = services.GetEventWithSender(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	event, err = services.EditMessage(event, data.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(event)
}

func deleteMessageEvent(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")
	messageId, _ := c.ParamsInt("messageId", 0)

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
	}

	var event models.Event
	if event, err = services.GetEventWithSender(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	event, err = services.DeleteMessage(event)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(event)
}
