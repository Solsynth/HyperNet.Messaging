package models

import "gorm.io/datatypes"

const (
	MessageTextType = "m.text"
)

type Message struct {
	BaseModel

	Uuid        string                    `json:"uuid"`
	Content     datatypes.JSONMap         `json:"content"`
	Type        string                    `json:"type"`
	Attachments datatypes.JSONSlice[uint] `json:"attachments"`
	Channel     Channel                   `json:"channel"`
	Sender      ChannelMember             `json:"sender"`
	ReplyID     *uint                     `json:"reply_id"`
	ReplyTo     *Message                  `json:"reply_to" gorm:"foreignKey:ReplyID"`
	ChannelID   uint                      `json:"channel_id"`
	SenderID    uint                      `json:"sender_id"`
}
