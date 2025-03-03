package models

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"gorm.io/datatypes"
)

const (
	EventMessageNew    = "messages.new"
	EventMessageEdit   = "messages.edit"
	EventMessageDelete = "messages.delete"
	EventSystemChanges = "system.changes"
)

type Event struct {
	cruda.BaseModel

	Uuid           string            `json:"uuid"`
	Body           datatypes.JSONMap `json:"body"`
	Type           string            `json:"type"`
	Channel        Channel           `json:"channel"`
	Sender         ChannelMember     `json:"sender"`
	QuoteEventID   *uint             `json:"quote_event_id,omitempty"`
	RelatedEventID *uint             `json:"related_event_id,omitempty"`
	ChannelID      uint              `json:"channel_id"`
	SenderID       uint              `json:"sender_id"`
}

// Event Payloads

type EventMessageBody struct {
	Text           string   `json:"text,omitempty"`
	KeyPair        string   `json:"keypair_id,omitempty"`
	Algorithm      string   `json:"algorithm,omitempty"`
	Attachments    []string `json:"attachments,omitempty"`
	QuoteEventID   *uint    `json:"quote_event,omitempty"`
	RelatedEventID *uint    `json:"related_event,omitempty"`
	RelatedUsers   []uint   `json:"related_users,omitempty"`
}
