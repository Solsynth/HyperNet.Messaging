package services

import (
	"context"
	"github.com/samber/lo"
	"time"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
)

func PushCommand(userId uint, task models.UnifiedCommand) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.H.GetDealerGrpcConn()
	_, _ = proto.NewStreamControllerClient(pc).PushStream(ctx, &proto.PushStreamRequest{
		UserId: lo.ToPtr(uint64(userId)),
		Body:   task.Marshal(),
	})
}

func PushCommandBatch(userId []uint64, task models.UnifiedCommand) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.H.GetDealerGrpcConn()
	_, _ = proto.NewStreamControllerClient(pc).PushStreamBatch(ctx, &proto.PushStreamBatchRequest{
		UserId: userId,
		Body:   task.Marshal(),
	})
}
