package models

import (
	"fmt"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
)

type ChannelType = uint8

const (
	ChannelTypeCommon = ChannelType(iota)
	ChannelTypeDirect
)

type Channel struct {
	cruda.BaseModel

	Alias       string          `json:"alias"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Members     []ChannelMember `json:"members"`
	Messages    []Event         `json:"messages"`
	Calls       []Call          `json:"calls"`
	Type        ChannelType     `json:"type"`
	AccountID   uint            `json:"account_id"`
	IsPublic    bool            `json:"is_public"`
	IsCommunity bool            `json:"is_community"`

	Realm   *authm.Realm `json:"realm" gorm:"-"`
	RealmID *uint        `json:"realm_id"`
}

func (v Channel) DisplayText() string {
	if v.Type == ChannelTypeDirect {
		return "DM"
	}
	if v.Realm != nil {
		return fmt.Sprintf("%s, %s", v.Name, v.Realm.Name)
	}
	return v.Name
}

type NotifyLevel = int8

const (
	NotifyLevelAll = NotifyLevel(iota)
	NotifyLevelMentioned
	NotifyLevelNone
)

type ChannelMember struct {
	cruda.BaseModel

	Name   string  `json:"name"`
	Nick   string  `json:"nick"`
	Avatar *string `json:"avatar"`

	ChannelID     uint        `json:"channel_id"`
	AccountID     uint        `json:"account_id"`
	Channel       Channel     `json:"channel"`
	Notify        NotifyLevel `json:"notify"`
	PowerLevel    int         `json:"power_level"`
	ReadingAnchor *int        `json:"reading_anchor"`

	Calls  []Call  `json:"calls" gorm:"foreignKey:FounderID"`
	Events []Event `json:"events" gorm:"foreignKey:SenderID"`
}
