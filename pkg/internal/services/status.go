package services

import (
	"fmt"
	"time"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cachekit"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type statusQueryCacheEntry struct {
	Target []uint64
	Data   any
}

func KgTypingStatusCache(channelId uint, userId uint) string {
	return fmt.Sprintf("chat-typing-status#%d@%d", userId, channelId)
}

func SetTypingStatus(channelId uint, userId uint) error {
	var broadcastTarget []uint64
	var data any

	hitCache := false
	if val, err := cachekit.Get[statusQueryCacheEntry](
		gap.Ca,
		KgTypingStatusCache(channelId, userId),
	); err == nil {
		broadcastTarget = val.Target
		data = val.Data
		hitCache = true
	}

	if !hitCache {
		var member models.ChannelMember
		if err := database.C.
			Where("account_id = ? AND channel_id = ?", userId, channelId).
			First(&member).Error; err != nil {
			return fmt.Errorf("channel member not found: %v", err)
		}

		var channel models.Channel
		if err := database.C.
			Preload("Members").
			Where("id = ?", channelId).
			First(&channel).Error; err != nil {
			return fmt.Errorf("channel not found: %v", err)
		}

		for _, item := range channel.Members {
			broadcastTarget = append(broadcastTarget, uint64(item.AccountID))
		}

		data = map[string]any{
			"user_id":    userId,
			"member_id":  member.ID,
			"channel_id": channelId,
			"member":     member,
			"channel":    channel,
		}

		// Cache queries
		cachekit.Set(
			gap.Ca,
			KgTypingStatusCache(channelId, userId),
			statusQueryCacheEntry{broadcastTarget, data},
			60*time.Minute,
			fmt.Sprintf("channel#%d", channelId),
		)
	}

	broadcastTarget = lo.Filter(broadcastTarget, func(item uint64, index int) bool {
		if !viper.GetBool("performance.passive_user_optimize") {
			// Leave this for backward compatibility
			return true
		}
		return CheckSubscribed(uint(item), channelId)
	})

	PushCommandBatch(broadcastTarget, nex.WebSocketPackage{
		Action:  "status.typing",
		Payload: data,
	})

	return nil
}
