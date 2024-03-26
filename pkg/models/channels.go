package models

type Channel struct {
	BaseModel

	Name        string          `json:"name"`
	Description string          `json:"description"`
	Members     []ChannelMember `json:"members"`
	AccountID   uint            `json:"account_id"`
}

type ChannelMember struct {
	BaseModel

	ChannelID uint    `json:"channel_id"`
	AccountID uint    `json:"account_id"`
	Channel   Channel `json:"channel"`
	Account   Account `json:"account"`
}
