package server

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
	"strings"
)

func listMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)
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
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("you need join the channel before you read the messages: %v", err))
	}

	count := services.CountMessage(channel)
	messages, err := services.ListMessage(channel, take, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  messages,
	})
}

func newMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var data struct {
		Uuid        string         `json:"uuid" validate:"required"`
		Type        string         `json:"type" validate:"required"`
		Content     map[string]any `json:"content"`
		Attachments []uint         `json:"attachments"`
		ReplyTo     *uint          `json:"reply_to"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	} else if len(data.Attachments) == 0 {
		if val, ok := data.Content["type"]; ok && val == models.MessageTextType {
			if val, ok := data.Content["value"].(string); ok && len(strings.TrimSpace(val)) == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "you cannot send an empty message")
			}
		}
	} else if len(data.Uuid) < 36 {
		return fiber.NewError(fiber.StatusBadRequest, "message uuid was not valid")
	}

	for _, attachment := range data.Attachments {
		if !services.CheckAttachmentByIDExists(attachment, "m.attachment") {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("attachment %d not found", attachment))
		}
	}

	var err error
	var channel models.Channel
	var member models.ChannelMember
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user, val.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	} else {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	}

	message := models.Message{
		Uuid:        data.Uuid,
		Content:     data.Content,
		Sender:      member,
		Channel:     channel,
		ChannelID:   channel.ID,
		SenderID:    member.ID,
		Attachments: data.Attachments,
		Type:        data.Type,
	}

	var replyTo models.Message
	if data.ReplyTo != nil {
		if err := database.C.Where("id = ?", data.ReplyTo).First(&replyTo).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("message to reply was not found: %v", err))
		} else {
			message.ReplyTo = &replyTo
			message.ReplyID = &replyTo.ID
		}
	}

	if message, err = services.NewMessage(message); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(message)
}

func editMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")
	messageId, _ := c.ParamsInt("messageId", 0)

	var data struct {
		Type        string         `json:"type" validate:"required"`
		Content     map[string]any `json:"content"`
		Attachments []uint         `json:"attachments"`
		ReplyTo     *uint          `json:"reply_to"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	for _, attachment := range data.Attachments {
		if !services.CheckAttachmentByIDExists(attachment, "m.attachment") {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("attachment %d not found", attachment))
		}
	}

	var err error
	var channel models.Channel
	var member models.ChannelMember
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user, val.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	} else {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	}

	var message models.Message
	if message, err = services.GetMessageWithPrincipal(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	message.Attachments = data.Attachments
	message.Content = data.Content
	message.Type = data.Type

	message, err = services.EditMessage(message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(message)
}

func deleteMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")
	messageId, _ := c.ParamsInt("messageId", 0)

	var err error
	var channel models.Channel
	var member models.ChannelMember
	if val, ok := c.Locals("realm").(models.Realm); ok {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user, val.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	} else {
		channel, member, err = services.GetAvailableChannelWithAlias(alias, user)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
	}

	var message models.Message
	if message, err = services.GetMessageWithPrincipal(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	message, err = services.DeleteMessage(message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(message)
}
