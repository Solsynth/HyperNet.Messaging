package services

import (
	"context"
	"fmt"
	"sync"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
)

type statusQueryCacheEntry struct {
	Account models.Account
	Member  models.ChannelMember
	Channel models.Channel
}

var statusQueryCacheLock sync.Mutex

// Map for caching typing status queries [channel id][user id]
var statusQueryCache map[uint]map[uint]statusQueryCacheEntry

func SetTypingStatus(channelId uint, userId uint) error {
	var account models.Account
	var member models.ChannelMember
	var channel models.Channel

	if channelLevel, ok := statusQueryCache[channelId]; ok {
		if entry, ok := channelLevel[userId]; ok {
			account = entry.Account
			member = entry.Member
			channel = entry.Channel
		}
	} else {
		if err := database.C.Where("external_id = ?", userId).First(&account).Error; err != nil {
			return fmt.Errorf("account not found: %v", err)
		}

		if err := database.C.
			Where("account_id = ? AND channel_id = ?", account.ID, channelId).
			First(&member).Error; err != nil {
			return fmt.Errorf("channel member not found: %v", err)
		} else {
			member.Account = account
		}

		if err := database.C.
			Preload("Members").
			Where("id = ?", channelId).
			First(&channel).Error; err != nil {
			return fmt.Errorf("channel not found: %v", err)
		}

		// Cache queries
		statusQueryCacheLock.Lock()
		if _, ok := statusQueryCache[channelId]; !ok {
			statusQueryCache[channelId] = make(map[uint]statusQueryCacheEntry)
		}
		statusQueryCache[channelId][userId] = statusQueryCacheEntry{account, member, channel}
		statusQueryCacheLock.Unlock()
	}

	var broadcastTarget []uint64
	for _, item := range channel.Members {
		if item.AccountID == member.AccountID {
			continue
		}
		broadcastTarget = append(broadcastTarget, uint64(item.AccountID))
	}

	sc := proto.NewStreamControllerClient(gap.H.GetDealerGrpcConn())
	_, err := sc.PushStreamBatch(context.Background(), &proto.PushStreamBatchRequest{
		UserId: broadcastTarget,
		Body: hyper.NetworkPackage{
			Action: "status.typing",
			Payload: map[string]any{
				"user_id":    userId,
				"member_id":  member.ID,
				"channel_id": channelId,
				"member":     member,
				"channel":    channel,
			},
		}.Marshal(),
	})

	return err
}
