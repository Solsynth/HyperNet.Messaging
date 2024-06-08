package models

type ChannelType = uint8

const (
	ChannelTypeCommon = ChannelType(iota)
	ChannelTypeDirect
)

type Channel struct {
	BaseModel

	Alias       string          `json:"alias"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Members     []ChannelMember `json:"members"`
	Messages    []Message       `json:"messages"`
	Calls       []Call          `json:"calls"`
	Type        ChannelType     `json:"type"`
	Account     Account         `json:"account"`
	AccountID   uint            `json:"account_id"`
	IsEncrypted bool            `json:"is_encrypted"`

	Realm   Realm `json:"realm"`
	RealmID *uint `json:"realm_id"`
}

type NotifyLevel = int8

const (
	NotifyLevelAll = NotifyLevel(iota)
	NotifyLevelMentioned
	NotifyLevelNone
)

type ChannelMember struct {
	BaseModel

	ChannelID  uint        `json:"channel_id"`
	AccountID  uint        `json:"account_id"`
	Nick       *string     `json:"nick"`
	Channel    Channel     `json:"channel"`
	Account    Account     `json:"account"`
	Notify     NotifyLevel `json:"notify"`
	PowerLevel int         `json:"power_level"`

	Calls    []Call    `json:"calls" gorm:"foreignKey:FounderID"`
	Messages []Message `json:"messages" gorm:"foreignKey:SenderID"`
}
