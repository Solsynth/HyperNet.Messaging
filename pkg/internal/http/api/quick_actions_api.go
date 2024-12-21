package api

import (
	"fmt"
	"strings"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/http/exts"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"
)

// quickReply is a simplified API for replying to a message
// It used in the iOS notification action and others
// It did not support all the features of the message event
// But it just works
func quickReply(c *fiber.Ctx) error {
	replyTk := c.Query("replyToken")
	if len(replyTk) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "reply token is required")
	}

	claims, err := services.ParseReplyToken(replyTk)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("reply token is invaild: %v", err))
	}

	channelId, _ := c.ParamsInt("channelId", 0)
	eventId, _ := c.ParamsInt("eventId", 0)

	if claims.EventID != uint(eventId) {
		return fiber.NewError(fiber.StatusBadRequest, "reply token is invaild, event id mismatch")
	}

	var data struct {
		Type string                  `json:"type" validate:"required"`
		Body models.EventMessageBody `json:"body"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else {
		data.Body.QuoteEventID = lo.ToPtr(uint(eventId))
	}

	data.Body.Text = strings.TrimSpace(data.Body.Text)
	if len(data.Body.Text) == 0 && len(data.Body.Attachments) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty message was not allowed")
	}

	channel, member, err := services.GetChannelIdentityWithID(uint(channelId), claims.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("channel / member not found: %v", err.Error()))
	}

	var parsed map[string]any
	raw, _ := jsoniter.Marshal(data.Body)
	_ = jsoniter.Unmarshal(raw, &parsed)

	event, err := services.NewEvent(models.Event{
		Uuid:           uuid.NewString(),
		Body:           parsed,
		Type:           data.Type,
		Sender:         member,
		Channel:        channel,
		QuoteEventID:   data.Body.QuoteEventID,
		RelatedEventID: data.Body.RelatedEventID,
		ChannelID:      channel.ID,
		SenderID:       member.ID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(event)
}
