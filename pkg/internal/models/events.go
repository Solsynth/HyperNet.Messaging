package models

import "gorm.io/datatypes"

const (
	EventMessageNew    = "messages.new"
	EventMessageEdit   = "messages.edit"
	EventMessageDelete = "messages.delete"
	EventSystemChanges = "system.changes"
)

type Event struct {
	BaseModel

	Uuid      string            `json:"uuid"`
	Body      datatypes.JSONMap `json:"body"`
	Type      string            `json:"type"`
	Channel   Channel           `json:"channel"`
	Sender    ChannelMember     `json:"sender"`
	ChannelID uint              `json:"channel_id"`
	SenderID  uint              `json:"sender_id"`
}

// Event Payloads

type EventMessageBody struct {
	Text         string `json:"text,omitempty"`
	Algorithm    string `json:"algorithm,omitempty"`
	Attachments  []uint `json:"attachments,omitempty"`
	QuoteEvent   uint   `json:"quote_event,omitempty"`
	RelatedEvent uint   `json:"related_event,omitempty"`
	RelatedUsers []uint `json:"related_users,omitempty"`
}
