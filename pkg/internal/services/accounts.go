package services

import (
	"context"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"github.com/samber/lo"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
)

func NotifyAccountMessagerBatch(users []authm.Account, notification *proto.NotifyRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	pc, err := gap.Nx.GetClientGrpcConn(nex.ServiceTypeAuth)
	if err != nil {
		return err
	}
	_, err = proto.NewNotifierClient(pc).NotifyUserBatch(ctx, &proto.NotifyUserBatchRequest{
		UserId: lo.Map(users, func(item authm.Account, idx int) uint64 {
			return uint64(item.ID)
		}),
		Notify: notification,
	})

	return err
}
