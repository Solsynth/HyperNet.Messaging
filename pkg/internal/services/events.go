package services

import (
	"fmt"
	"strings"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func CountEvent(channel models.Channel) int64 {
	var count int64
	if err := database.C.Where(models.Event{
		ChannelID: channel.ID,
	}).Model(&models.Event{}).Count(&count).Error; err != nil {
		return 0
	} else {
		return count
	}
}

func ListEvent(channel models.Channel, take int, offset int) ([]models.Event, error) {
	if take > 100 {
		take = 100
	}

	var events []models.Event
	if err := database.C.
		Where(models.Event{
			ChannelID: channel.ID,
		}).Limit(take).Offset(offset).
		Order("created_at DESC").
		Preload("Sender").
		Preload("Sender.Account").
		Find(&events).Error; err != nil {
		return events, err
	} else {
		return events, nil
	}
}

func GetEvent(channel models.Channel, id uint) (models.Event, error) {
	var event models.Event
	if err := database.C.
		Where(models.Event{
			BaseModel: models.BaseModel{ID: id},
			ChannelID: channel.ID,
		}).
		Preload("Sender").
		Preload("Sender.Account").
		First(&event).Error; err != nil {
		return event, err
	} else {
		return event, nil
	}
}

func GetEventWithSender(channel models.Channel, member models.ChannelMember, id uint) (models.Event, error) {
	var event models.Event
	if err := database.C.Where(models.Event{
		BaseModel: models.BaseModel{ID: id},
		ChannelID: channel.ID,
		SenderID:  member.ID,
	}).First(&event).Error; err != nil {
		return event, err
	} else {
		return event, nil
	}
}

func NewEvent(event models.Event) (models.Event, error) {
	var members []models.ChannelMember
	if err := database.C.Save(&event).Error; err != nil {
		return event, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: event.ChannelID,
	}).Preload("Account").Find(&members).Error; err != nil {
		// Couldn't get channel members, skip notifying
		return event, nil
	}

	channel := event.Channel
	event, _ = GetEvent(event.Channel, event.ID)
	idxList := lo.Map(members, func(item models.ChannelMember, index int) uint64 {
		return uint64(item.AccountID)
	})
	PushCommandBatch(idxList, models.UnifiedCommand{
		Action:  "events.new",
		Payload: event,
	})

	if strings.HasPrefix(event.Type, "messages") {
		event.Channel = channel
		NotifyMessageEvent(members, event)
	}

	return event, nil
}

func NotifyMessageEvent(members []models.ChannelMember, event models.Event) {
	var body models.EventMessageBody
	raw, _ := jsoniter.Marshal(event.Body)
	_ = jsoniter.Unmarshal(raw, &body)

	var pendingUsers []models.Account

	for _, member := range members {
		if member.ID != event.SenderID {
			switch member.Notify {
			case models.NotifyLevelNone:
				continue
			case models.NotifyLevelMentioned:
				if len(body.RelatedUsers) == 0 || !lo.Contains(body.RelatedUsers, member.AccountID) {
					continue
				}
			default:
				break
			}

			pendingUsers = append(pendingUsers, member.Account)
		}
	}

	var displayText string
	if body.Algorithm == "plain" {
		displayText = body.Text
	}

	if len(displayText) == 0 {
		displayText = fmt.Sprintf("%d attachment(s)", len(body.Attachments))
	}

	err := NotifyAccountMessagerBatch(
		pendingUsers,
		&proto.NotifyRequest{
			Topic:  "messaging.message",
			Title:  fmt.Sprintf("%s in %s", event.Sender.Account.Nick, event.Channel.DisplayText()),
			Body:   displayText,
			Avatar: &event.Sender.Account.Avatar,
			Metadata: EncodeJSONBody(map[string]any{
				"user_id":    event.Sender.Account.ExternalID,
				"user_name":  event.Sender.Account.Name,
				"user_nick":  event.Sender.Account.Nick,
				"channel_id": event.ChannelID,
			}),
			IsRealtime:  true,
			IsForcePush: false,
		},
	)
	if err != nil {
		log.Warn().Err(err).Msg("An error occurred when trying notify user.")
	}
}

func EditEvent(event models.Event) (models.Event, error) {
	if err := database.C.Save(&event).Error; err != nil {
		return event, err
	}
	return event, nil
}

func DeleteEvent(event models.Event) (models.Event, error) {
	if err := database.C.Delete(&event).Error; err != nil {
		return event, err
	}
	return event, nil
}
