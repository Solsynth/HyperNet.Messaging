package services

import (
	"context"
	"strconv"
	"time"

	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"github.com/rs/zerolog/log"

	"github.com/samber/lo"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/proto"
)

func PushCommand(userId uint, task nex.WebSocketPackage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	_, err := proto.NewStreamServiceClient(pc).PushStream(ctx, &proto.PushStreamRequest{
		UserId: lo.ToPtr(uint64(userId)),
		Body:   task.Marshal(),
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to push websocket command to nexus...")
	}
}

func PushCommandBatch(userId []uint64, task nex.WebSocketPackage) []uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pc := gap.Nx.GetNexusGrpcConn()
	resp, err := proto.NewStreamServiceClient(pc).PushStreamBatch(ctx, &proto.PushStreamBatchRequest{
		UserId: userId,
		Body:   task.Marshal(),
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to push websocket command to nexus in batches...")
	}

	return lo.Map(resp.GetSuccessList(), func(item string, _ int) uint64 {
		val, _ := strconv.ParseUint(item, 10, 64)
		return val
	})
}
