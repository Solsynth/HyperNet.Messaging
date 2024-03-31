package models

import "gorm.io/datatypes"

type MessageType = uint8

const (
	MessageTypeText = MessageType(iota)
	MessageTypeAudio
)

type Message struct {
	BaseModel

	Content     string            `json:"content"`
	Metadata    datatypes.JSONMap `json:"metadata"`
	Type        MessageType       `json:"type"`
	Attachments []Attachment      `json:"attachments"`
	Channel     Channel           `json:"channel"`
	Sender      ChannelMember     `json:"sender"`
	ReplyID     *uint             `json:"reply_id"`
	ReplyTo     *Message          `json:"reply_to" gorm:"foreignKey:ReplyID"`
	ChannelID   uint              `json:"channel_id"`
	SenderID    uint              `json:"sender_id"`
}
