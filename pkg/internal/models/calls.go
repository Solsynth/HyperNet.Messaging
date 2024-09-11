package models

import (
	"time"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"github.com/livekit/protocol/livekit"
)

type Call struct {
	hyper.BaseModel

	EndedAt *time.Time `json:"ended_at"`

	ExternalID string        `json:"external_id"`
	FounderID  uint          `json:"founder_id"`
	ChannelID  uint          `json:"channel_id"`
	Founder    ChannelMember `json:"founder"`
	Channel    Channel       `json:"channel"`

	Participants []*livekit.ParticipantInfo `json:"participants" gorm:"-"`
}
