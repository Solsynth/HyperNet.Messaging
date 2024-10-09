package services

import (
	"context"
	"fmt"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	localCache "git.solsynth.dev/hydrogen/messaging/pkg/internal/cache"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"
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
		var account models.Account
		if err := database.C.Where("external_id = ?", userId).First(&account).Error; err != nil {
			return fmt.Errorf("account not found: %v", err)
		}

		var member models.ChannelMember
		if err := database.C.
			Where("account_id = ? AND channel_id = ?", account.ID, channelId).
			First(&member).Error; err != nil {
			return fmt.Errorf("channel member not found: %v", err)
		} else {
			member.Account = account
		}

		var channel models.Channel
		if err := database.C.
			Preload("Members").
			Preload("Members.Account").
			Where("id = ?", channelId).
			First(&channel).Error; err != nil {
			return fmt.Errorf("channel not found: %v", err)
		}

		for _, item := range channel.Members {
			broadcastTarget = append(broadcastTarget, uint64(item.Account.ID))
		}

		data = map[string]any{
			"user_id":    userId,
			"member_id":  member.ID,
			"channel_id": channelId,
			"member":     member,
			"channel":    channel,
		}

		// Cache queries
		cacheManager := cache.New[any](localCache.S)
		marshal := marshaler.New(cacheManager)
		contx := context.Background()

		marshal.Set(
			contx,
			GetTypingStatusQueryCacheKey(channelId, userId),
			statusQueryCacheEntry{broadcastTarget, data},
			store.WithTags([]string{"typing-status-query", fmt.Sprintf("channel#%d", channelId), fmt.Sprintf("user#%d", userId)}),
		)
	}

	sc := proto.NewStreamControllerClient(gap.H.GetDealerGrpcConn())
	_, err := sc.PushStreamBatch(context.Background(), &proto.PushStreamBatchRequest{
		UserId: broadcastTarget,
		Body: hyper.NetworkPackage{
			Action:  "status.typing",
			Payload: data,
		}.Marshal(),
	})

	return err
}
