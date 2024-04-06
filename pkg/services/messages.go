package services

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/rs/zerolog/log"
)

func CountMessage(channel models.Channel) int64 {
	var count int64
	if err := database.C.Where(models.Message{
		ChannelID: channel.ID,
	}).Model(&models.Message{}).Count(&count).Error; err != nil {
		return 0
	} else {
		return count
	}
}

func ListMessage(channel models.Channel, take int, offset int) ([]models.Message, error) {
	if take > 100 {
		take = 100
	}

	var messages []models.Message
	if err := database.C.
		Where(models.Message{
			ChannelID: channel.ID,
		}).Limit(take).Offset(offset).
		Order("created_at DESC").
		Preload("Attachments").
		Preload("ReplyTo").
		Preload("ReplyTo.Sender").
		Preload("ReplyTo.Sender.Account").
		Preload("Sender").
		Preload("Sender.Account").
		Find(&messages).Error; err != nil {
		return messages, err
	} else {
		return messages, nil
	}
}

func GetMessage(channel models.Channel, id uint) (models.Message, error) {
	var message models.Message
	if err := database.C.
		Where(models.Message{
			BaseModel: models.BaseModel{ID: id},
			ChannelID: channel.ID,
		}).
		Preload("ReplyTo").
		Preload("ReplyTo.Sender").
		Preload("ReplyTo.Sender.Account").
		Preload("Attachments").
		Preload("Sender").
		Preload("Sender.Account").
		First(&message).Error; err != nil {
		return message, err
	} else {
		return message, nil
	}
}

func GetMessageWithPrincipal(channel models.Channel, member models.ChannelMember, id uint) (models.Message, error) {
	var message models.Message
	if err := database.C.Where(models.Message{
		BaseModel: models.BaseModel{ID: id},
		ChannelID: channel.ID,
		SenderID:  member.ID,
	}).First(&message).Error; err != nil {
		return message, err
	} else {
		return message, nil
	}
}

func NewMessage(message models.Message) (models.Message, error) {
	var members []models.ChannelMember
	if err := database.C.Save(&message).Error; err != nil {
		return message, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: message.ChannelID,
	}).Preload("Account").Find(&members).Error; err == nil {
		channel := message.Channel
		message, _ = GetMessage(message.Channel, message.ID)
		for _, member := range members {
			if member.ID != message.Sender.ID {
				// TODO Check the mentioned status
				if member.Notify == models.NotifyLevelAll {
					err = NotifyAccount(member.Account,
						fmt.Sprintf("New Message #%s", channel.Alias),
						fmt.Sprintf("%s: %s", message.Sender.Account.Name, message.Content),
						true,
					)
					if err != nil {
						log.Warn().Err(err).Msg("An error occurred when trying notify user.")
					}
				}
			}
			PushCommand(member.AccountID, models.UnifiedCommand{
				Action:  "messages.new",
				Payload: message,
			})
		}
	}

	return message, nil
}

func EditMessage(message models.Message) (models.Message, error) {
	var members []models.ChannelMember
	if err := database.C.Save(&message).Error; err != nil {
		return message, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: message.ChannelID,
	}).Find(&members).Error; err == nil {
		message, _ = GetMessage(models.Channel{
			BaseModel: models.BaseModel{ID: message.Channel.ID},
		}, message.ID)
		for _, member := range members {
			PushCommand(member.AccountID, models.UnifiedCommand{
				Action:  "messages.update",
				Payload: message,
			})
		}
	}

	return message, nil
}

func DeleteMessage(message models.Message) (models.Message, error) {
	prev, _ := GetMessage(models.Channel{
		BaseModel: models.BaseModel{ID: message.Channel.ID},
	}, message.ID)

	var members []models.ChannelMember
	if err := database.C.Delete(&message).Error; err != nil {
		return message, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: message.ChannelID,
	}).Find(&members).Error; err == nil {
		for _, member := range members {
			PushCommand(member.AccountID, models.UnifiedCommand{
				Action:  "messages.burnt",
				Payload: prev,
			})
		}
	}

	return message, nil
}
