package services

import (
	"fmt"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
)

func GetDirectChannelByUser(user authm.Account, other authm.Account) (models.Channel, error) {
	memberTable := "channel_members"
	channelTable := "channels"

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
