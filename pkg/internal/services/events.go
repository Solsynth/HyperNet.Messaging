package services

import (
	"fmt"
	"strings"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	"git.solsynth.dev/hypernet/pusher/pkg/pushkit"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
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
		Find(&events).Error; err != nil {
		return events, err
	} else {
		return events, nil
	}
}

func GetEvent(channelId uint, id uint) (models.Event, error) {
	var event models.Event
	if err := database.C.
		Where("id = ? AND channel_id = ?", id, channelId).
		Preload("Sender").
		First(&event).Error; err != nil {
		return event, err
	} else {
		return event, nil
	}
}

func GetEventWithSender(channel models.Channel, member models.ChannelMember, id uint) (models.Event, error) {
	var event models.Event
	if err := database.C.Where(models.Event{
		BaseModel: cruda.BaseModel{ID: id},
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
	}).Find(&members).Error; err != nil {
		// Couldn't get channel members, skip notifying
		log.Warn().Err(err).Msg("Failed to fetch members, the notifying of new event was terminated...")
		return event, nil
	}

	event, err := GetEvent(event.ChannelID, event.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch event, the notifying of new event was terminated...")
		return event, err
	}
	idxList := lo.Map(members, func(item models.ChannelMember, index int) uint64 {
		return uint64(item.AccountID)
	})
	_ = PushCommandBatch(idxList, nex.WebSocketPackage{
		Action:  "events.new",
		Payload: event,
	})

	if strings.HasPrefix(event.Type, "messages") {
		event.Channel, _ = GetChannel(event.ChannelID)
		if event.Channel.RealmID != nil {
			realm, err := authkit.GetRealm(gap.Nx, *event.Channel.RealmID)
			if err == nil {
				event.Channel.Realm = &realm
			}
		}
		go NotifyMessageEvent(members, event)
	}

	return event, nil
}

func NotifyMessageEvent(members []models.ChannelMember, event models.Event) {
	var body models.EventMessageBody
	raw, _ := jsoniter.Marshal(event.Body)
	_ = jsoniter.Unmarshal(raw, &body)

	var pendingUsers []uint64
	var mentionedUsers []uint64

	for _, member := range members {
		if member.ID != event.SenderID {
			switch member.Notify {
			case models.NotifyLevelNone:
				continue
			case models.NotifyLevelMentioned:
				if len(body.RelatedUsers) != 0 && lo.Contains(body.RelatedUsers, member.AccountID) {
					mentionedUsers = append(mentionedUsers, uint64(member.AccountID))
				}
				continue
			default:
				break
			}

			if lo.Contains(body.RelatedUsers, member.AccountID) {
				mentionedUsers = append(mentionedUsers, uint64(member.AccountID))
			} else {
				pendingUsers = append(pendingUsers, uint64(member.AccountID))
			}
		}
	}

	var displayText string
	var displaySubtitle string
	switch event.Type {
	case models.EventMessageNew:
		if body.Algorithm == "plain" {
			displayText = body.Text
		}
	case models.EventMessageEdit:
		displaySubtitle = "Edited a message"
		if body.Algorithm == "plain" {
			displayText = body.Text
		}
	case models.EventMessageDelete:
		displayText = "Deleted a message"
	}

	if len(displayText) == 0 {
		if len(displayText) == 1 {
			displayText = fmt.Sprintf("%d file", len(body.Attachments))
		} else {
			displayText = fmt.Sprintf("%d files", len(body.Attachments))
		}
	} else if len(body.Attachments) > 0 {
		if len(displayText) == 1 {
			displayText += fmt.Sprintf(" (%d file)", len(body.Attachments))
		} else {
			displayText += fmt.Sprintf(" (%d files)", len(body.Attachments))
		}
	}

	user, err := authkit.GetUser(gap.Nx, event.Sender.AccountID)
	if err == nil {
		event.Sender.Avatar = user.Avatar
		if len(event.Sender.Nick) == 0 {
			event.Sender.Nick = user.Nick
		}
	}

	displayTitle := fmt.Sprintf("%s (%s)", event.Sender.Nick, event.Channel.DisplayText())

	metadata := map[string]any{
		"avatar":     event.Sender.Avatar,
		"user_id":    event.Sender.AccountID,
		"user_name":  event.Sender.Name,
		"user_nick":  event.Sender.Nick,
		"channel_id": event.ChannelID,
		"event_id":   event.ID,
	}

	if len(pendingUsers) > 0 {
		log.Debug().
			Uint("event_id", event.ID).
			Str("title", displayTitle).
			Int("count", len(pendingUsers)).
			Msg("Notifying new event...")

		for _, pendingUser := range pendingUsers {
			replyToken, err := CreateReplyToken(event.ID, uint(pendingUser))
			if err != nil {
				log.Warn().Err(err).Msg("An error occurred when trying create reply token.")
				continue
			}
			metadata["reply_token"] = replyToken
			err = authkit.NotifyUser(
				gap.Nx,
				pendingUser,
				pushkit.Notification{
					Topic:    "messaging.message",
					Title:    displayTitle,
					Subtitle: displaySubtitle,
					Body:     displayText,
					Metadata: metadata,
					Priority: 10,
				},
				true,
			)
			if err != nil {
				log.Warn().Err(err).Msg("An error occurred when trying notify user.")
			}
		}
	}

	if len(mentionedUsers) > 0 {
		if len(displaySubtitle) > 0 {
			displaySubtitle += ", and mentioned you"
		} else {
			displaySubtitle = "Mentioned you"
		}

		log.Debug().
			Uint("event_id", event.ID).
			Str("title", displayTitle).
			Int("count", len(mentionedUsers)).
			Msg("Notifying new event...")

		for _, mentionedUser := range mentionedUsers {
			replyToken, err := CreateReplyToken(event.ID, uint(mentionedUser))
			if err != nil {
				log.Warn().Err(err).Msg("An error occurred when trying create reply token.")
				continue
			}
			metadata["reply_token"] = replyToken
			err = authkit.NotifyUser(
				gap.Nx,
				mentionedUser,
				pushkit.Notification{
					Topic:    "messaging.message",
					Title:    displayTitle,
					Subtitle: displaySubtitle,
					Body:     displayText,
					Metadata: metadata,
					Priority: 10,
				},
				true,
			)
			if err != nil {
				log.Warn().Err(err).Msg("An error occurred when trying notify user.")
			}
		}
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
