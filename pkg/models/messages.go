package models

type Message struct {
	BaseModel

	Content     []byte        `json:"content"`
	Type        string        `json:"type"`
	Attachments []Attachment  `json:"attachments"`
	Channel     Channel       `json:"channel"`
	Sender      ChannelMember `json:"sender"`
	ReplyID     *uint         `json:"reply_id"`
	ReplyTo     *Message      `json:"reply_to" gorm:"foreignKey:ReplyID"`
	ChannelID   uint          `json:"channel_id"`
	SenderID    uint          `json:"sender_id"`
}
