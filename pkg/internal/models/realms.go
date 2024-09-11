package models

import "git.solsynth.dev/hydrogen/dealer/pkg/hyper"

// Realm profiles basically fetched from Hydrogen.Passport
// But cache at here for better usage and database relations
type Realm struct {
	hyper.BaseModel

	Alias       string    `json:"alias"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Channels    []Channel `json:"channels"`
	IsPublic    bool      `json:"is_public"`
	IsCommunity bool      `json:"is_community"`
	ExternalID  uint      `json:"external_id"`
}
