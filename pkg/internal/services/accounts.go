package services

import (
	"context"
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"git.solsynth.dev/hydrogen/passport/pkg/proto"
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

	pc, err := gap.H.DiscoverServiceGRPC("Hydrogen.Passport")
	if err != nil {
		return nil, err
	}
	return proto.NewFriendshipsClient(pc).GetFriendship(ctx, &proto.FriendshipTwoSideLookupRequest{
		AccountId: uint64(user.ExternalID),
		RelatedId: uint64(related.ExternalID),
		Status:    uint32(status),
	})
}

func NotifyAccountMessager(user models.Account, t, s, c string, realtime bool, forcePush bool, links ...*proto.NotifyLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pc, err := gap.H.DiscoverServiceGRPC("Hydrogen.Passport")
	if err != nil {
		return err
	}
	_, err = proto.NewNotifyClient(pc).NotifyUser(ctx, &proto.NotifyRequest{
		ClientId:     viper.GetString("passport.client_id"),
		ClientSecret: viper.GetString("passport.client_secret"),
		Type:         fmt.Sprintf("messaging.%s", t),
		Subject:      s,
		Content:      c,
		Links:        links,
		RecipientId:  uint64(user.ExternalID),
		IsRealtime:   realtime,
		IsForcePush:  forcePush,
	})

	return err
}
