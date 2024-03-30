package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
)

func NewTextMessage(content string, sender models.ChannelMember, channel models.Channel) (models.Message, error) {
	message := models.Message{
		Content:   content,
		Metadata:  nil,
		ChannelID: channel.ID,
		SenderID:  sender.ID,
		Type:      models.MessageTypeText,
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
