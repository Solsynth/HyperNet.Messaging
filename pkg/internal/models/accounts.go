package models

// Account profiles basically fetched from Hydrogen.Identity
// But cached at here for better usage
// At the same time, this model can make relations between local models
type Account struct {
	BaseModel

	Name         string    `json:"name"`
	Nick         string    `json:"nick"`
	Avatar       string    `json:"avatar"`
	Banner       string    `json:"banner"`
	Description  string    `json:"description"`
	EmailAddress string    `json:"email_address"`
	PowerLevel   int       `json:"power_level"`
	Channels     []Channel `json:"channels"`
	ExternalID   uint      `json:"external_id"`
}
