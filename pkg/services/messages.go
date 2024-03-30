package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
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
	if err := database.C.Where(models.Message{
		ChannelID: channel.ID,
	}).Limit(take).Offset(offset).Find(&messages).Error; err != nil {
		return messages, err
	} else {
		return messages, nil
	}
}

func NewTextMessage(content string, sender models.ChannelMember, channel models.Channel, attachments ...models.Attachment) (models.Message, error) {
	message := models.Message{
		Content:     content,
		Metadata:    nil,
		ChannelID:   channel.ID,
		SenderID:    sender.ID,
		Attachments: attachments,
		Type:        models.MessageTypeText,
	}

	var members []models.ChannelMember
	if err := database.C.Save(&message).Error; err != nil {
		return message, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: channel.ID,
	}).Find(&members).Error; err == nil {
		for _, member := range members {
			PushCommand(member.ID, models.UnifiedCommand{
				Action:  "messages.new",
				Payload: message,
			})
		}
	}

	return message, nil
}
