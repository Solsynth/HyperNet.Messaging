package models

import "gorm.io/datatypes"

type MessageType = uint8

const (
	MessageTypeText = MessageType(iota)
	MessageTypeAudio
	MessageTypeFile
)

type Message struct {
	BaseModel

	Content     string            `json:"content"`
	Metadata    datatypes.JSONMap `json:"metadata"`
	Type        MessageType       `json:"type"`
	Attachments []Attachment      `json:"attachments"`
	Channel     Channel           `json:"channel"`
	Sender      Account           `json:"sender"`
	ChannelID   uint              `json:"channel_id"`
	SenderID    uint              `json:"sender_id"`
}
