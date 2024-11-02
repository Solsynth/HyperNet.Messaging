package models

import "git.solsynth.dev/hydrogen/dealer/pkg/hyper"

// Realm profiles basically fetched from Hypernet.Passport
// But cache at here for better usage and database relations
type Realm struct {
	hyper.BaseRealm

	Channels []Channel `json:"channels"`
}
