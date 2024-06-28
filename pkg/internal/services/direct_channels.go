package services

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/spf13/viper"
)

func GetDirectChannelByUser(user models.Account, other models.Account) (models.Channel, error) {
	memberTable := fmt.Sprintf("%schannel_members", viper.GetString("database.prefix"))
	channelTable := fmt.Sprintf("%schannels", viper.GetString("database.prefix"))

	var channel models.Channel
	if err := database.C.Preload("Members").
		Where("type = ?", models.ChannelTypeDirect).
		Joins(fmt.Sprintf("JOIN %s cm1 ON cm1.channel_id = %s.id AND cm1.account_id = ?", memberTable, channelTable), user.ID).
		Joins(fmt.Sprintf("JOIN %s cm2 ON cm2.channel_id = %s.id AND cm2.account_id = ?", memberTable, channelTable), other.ID).
		First(&channel).Error; err != nil {
		return channel, err
	} else {
		return channel, nil
	}
}
