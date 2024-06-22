package services

import (
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/spf13/viper"
)

var Lk *lksdk.RoomServiceClient

func SetupLiveKit() {
	host := "https://" + viper.GetString("calling.endpoint")

	Lk = lksdk.NewRoomServiceClient(
		host,
		viper.GetString("calling.api_key"),
		viper.GetString("calling.api_secret"),
	)
}
