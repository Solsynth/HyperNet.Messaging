package grpc

import (
	"context"
	"fmt"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/server/exts"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/services"
	jsoniter "github.com/json-iterator/go"
)

func (v *Server) EmitStreamEvent(_ context.Context, in *proto.StreamEventRequest) (*proto.StreamEventResponse, error) {
	sc := proto.NewStreamControllerClient(gap.H.GetDealerGrpcConn())

	switch in.GetEvent() {
	case "status.typing":
		var data struct {
			ChannelID uint `json:"channel_id" validate:"required"`
		}

		err := jsoniter.Unmarshal(in.GetPayload(), &data)
		if err == nil {
			err = exts.ValidateStruct(data)
		}
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: &in.ClientId,
				Body: hyper.NetworkPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable parse payload: %v", err),
				}.Marshal(),
			})
		}

		err = services.SetTypingStatus(data.ChannelID, uint(in.GetUserId()))
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: &in.ClientId,
				Body: hyper.NetworkPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable boardcast status: %v", err),
				}.Marshal(),
			})
		}
	}

	return &proto.StreamEventResponse{}, nil
}
