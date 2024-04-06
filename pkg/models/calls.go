package models

import "time"

type CallProvider = string

const (
	CallProviderJitsi = "jitsi"
)

type Call struct {
	BaseModel

	Provider string     `json:"provider"`
	EndedAt  *time.Time `json:"ended_at"`

	ExternalID string        `json:"external_id"`
	FounderID  uint          `json:"founder_id"`
	ChannelID  uint          `json:"channel_id"`
	Founder    ChannelMember `json:"founder"`
	Channel    Channel       `json:"channel"`
}
