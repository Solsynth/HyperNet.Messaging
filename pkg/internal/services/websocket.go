package services

import (
	"context"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"time"

	"github.com/samber/lo"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/proto"
)

func PushCommand(userId uint, task nex.WebSocketPackage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	_, _ = proto.NewStreamServiceClient(pc).PushStream(ctx, &proto.PushStreamRequest{
		UserId: lo.ToPtr(uint64(userId)),
		Body:   task.Marshal(),
	})
}

func PushCommandBatch(userId []uint64, task nex.WebSocketPackage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	_, _ = proto.NewStreamServiceClient(pc).PushStreamBatch(ctx, &proto.PushStreamBatchRequest{
		UserId: userId,
		Body:   task.Marshal(),
	})
}
