package services

import (
	"context"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg/grpc"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/passport/pkg/grpc/proto"
	"github.com/spf13/viper"
)

func GetAccountFriend(userId, relatedId uint, status int) (*proto.FriendshipResponse, error) {
	var user models.Account
	if err := database.C.Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}
	var related models.Account
	if err := database.C.Where("id = ?", relatedId).First(&related).Error; err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return grpc.Friendships.GetFriendship(ctx, &proto.FriendshipTwoSideLookupRequest{
		AccountId: uint64(user.ExternalID),
		RelatedId: uint64(related.ExternalID),
		Status:    uint32(status),
	})
}

func NotifyAccountMessager(user models.Account, subject, content string, realtime bool, forcePush bool, links ...*proto.NotifyLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := grpc.Notify.NotifyUser(ctx, &proto.NotifyRequest{
		ClientId:     viper.GetString("passport.client_id"),
		ClientSecret: viper.GetString("passport.client_secret"),
		Type:         "notifications.messaging",
		Subject:      subject,
		Content:      content,
		Links:        links,
		RecipientId:  uint64(user.ExternalID),
		IsRealtime:   realtime,
		IsForcePush:  forcePush,
	})

	return err
}
