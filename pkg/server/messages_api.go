package server

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
)

func listMessage(c *fiber.Ctx) error {
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)
	alias := c.Params("channel")

	channel, err := services.GetChannelWithAlias(alias)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
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

func newTextMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var data struct {
		Content     string              `json:"content" validate:"required"`
		Attachments []models.Attachment `json:"attachments"`
		ReplyTo     *uint               `json:"reply_to"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	channel, member, err := services.GetAvailableChannelWithAlias(alias, user)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	message := models.Message{
		Content:     data.Content,
		Metadata:    nil,
		Sender:      member,
		Channel:     channel,
		ChannelID:   channel.ID,
		SenderID:    member.ID,
		Attachments: data.Attachments,
		Type:        models.MessageTypeText,
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
		Content     string              `json:"content" validate:"required"`
		Attachments []models.Attachment `json:"attachments"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	var message models.Message
	if channel, member, err := services.GetAvailableChannelWithAlias(alias, user); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if message, err = services.GetMessageWithPrincipal(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	message.Content = data.Content
	message.Attachments = data.Attachments

	message, err := services.EditMessage(message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(message)
}

func deleteMessage(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")
	messageId, _ := c.ParamsInt("messageId", 0)

	var message models.Message
	if channel, member, err := services.GetAvailableChannelWithAlias(alias, user); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if message, err = services.GetMessageWithPrincipal(channel, member, uint(messageId)); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	message, err := services.DeleteMessage(message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(message)
}
