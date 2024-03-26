package services

import (
	"context"
	"time"

	"git.solsynth.dev/hydrogen/identity/pkg/grpc/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/grpc"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/spf13/viper"
)

func NotifyAccount(user models.Account, subject, content string, links ...*proto.NotifyLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err := grpc.Notify.NotifyUser(ctx, &proto.NotifyRequest{
		ClientId:     viper.GetString("identity.client_id"),
		ClientSecret: viper.GetString("identity.client_secret"),
		Subject:      subject,
		Content:      content,
		Links:        links,
		RecipientId:  uint64(user.ID),
		IsImportant:  false,
	})

	return err
}
