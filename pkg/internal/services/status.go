package services

import (
	"context"
	"fmt"

	localCache "git.solsynth.dev/hypernet/messaging/pkg/internal/cache"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type statusQueryCacheEntry struct {
	Target []uint64
	Data   any
}

func GetTypingStatusQueryCacheKey(channelId uint, userId uint) string {
	return fmt.Sprintf("typing-status-query#%d;%d", channelId, userId)
}

func SetTypingStatus(channelId uint, userId uint) error {
	var broadcastTarget []uint64
	var data any

	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	hitCache := false
	if val, err := marshal.Get(contx, GetTypingStatusQueryCacheKey(channelId, userId), new(statusQueryCacheEntry)); err == nil {
		entry := val.(*statusQueryCacheEntry)
		broadcastTarget = entry.Target
		data = entry.Data
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
		_ = marshal.Set(
			contx,
			GetTypingStatusQueryCacheKey(channelId, userId),
			statusQueryCacheEntry{broadcastTarget, data},
			store.WithTags([]string{"typing-status-query", fmt.Sprintf("channel#%d", channelId), fmt.Sprintf("user#%d", userId)}),
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
