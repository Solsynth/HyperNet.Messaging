package services

import (
	"context"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"time"

	"github.com/samber/lo"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
)

func PushCommand(userId uint, task nex.WebSocketPackage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	_, _ = proto.NewStreamControllerClient(pc).PushStream(ctx, &proto.PushStreamRequest{
		UserId: lo.ToPtr(uint64(userId)),
		Body:   task.Marshal(),
	})
}

func PushCommandBatch(userId []uint64, task nex.WebSocketPackage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	_, _ = proto.NewStreamControllerClient(pc).PushStreamBatch(ctx, &proto.PushStreamBatchRequest{
		UserId: userId,
		Body:   task.Marshal(),
	})
}
