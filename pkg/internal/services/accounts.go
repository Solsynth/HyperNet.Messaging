package services

import (
	"context"
	"fmt"
	"time"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
)

func CheckUserPerm(userId, otherId uint, key string, val any) error {
	var user models.Account
	if err := database.C.Where("id = ?", userId).First(&user).Error; err != nil {
		return fmt.Errorf("account not found: %v", err)
	}
	var other models.Account
	if err := database.C.Where("id = ?", otherId).First(&other).Error; err != nil {
		return fmt.Errorf("other not found: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	encodedData, _ := jsoniter.Marshal(val)

	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return err
	}
	out, err := proto.NewAuthClient(pc).EnsureUserPermGranted(ctx, &proto.CheckUserPermRequest{
		UserId:  uint64(user.ID),
		OtherId: uint64(other.ID),
		Key:     key,
		Value:   encodedData,
	})

	if err != nil {
		return err
	} else if !out.IsValid {
		return fmt.Errorf("missing permission: %v", key)
	}

	return nil
}

func NotifyAccountMessager(user models.Account, notification *proto.NotifyRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return err
	}
	_, err = proto.NewNotifierClient(pc).NotifyUser(ctx, &proto.NotifyUserRequest{
		UserId: uint64(user.ID),
		Notify: notification,
	})

	return err
}

func NotifyAccountMessagerBatch(users []models.Account, notification *proto.NotifyRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return err
	}
	_, err = proto.NewNotifierClient(pc).NotifyUserBatch(ctx, &proto.NotifyUserBatchRequest{
		UserId: lo.Map(users, func(item models.Account, idx int) uint64 {
			return uint64(item.ID)
		}),
		Notify: notification,
	})

	return err
}
