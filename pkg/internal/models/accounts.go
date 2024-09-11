package models

import "git.solsynth.dev/hydrogen/dealer/pkg/hyper"

type Account struct {
	hyper.BaseUser

	Channels []Channel `json:"channels"`
}
