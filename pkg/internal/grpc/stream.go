package grpc

import (
	"context"
	"fmt"
	"strings"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/web/exts"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/nexus/pkg/proto"
	jsoniter "github.com/json-iterator/go"
)

func (v *Server) PushStream(_ context.Context, request *proto.PushStreamRequest) (*proto.PushStreamResponse, error) {
	sc := proto.NewStreamServiceClient(gap.Nx.GetNexusGrpcConn())

	var in nex.WebSocketPackage
	if err := jsoniter.Unmarshal(request.GetBody(), &in); err != nil {
		return nil, err
	}

	switch in.Action {
	case "status.typing":
		var data struct {
			ChannelID uint `json:"channel_id" validate:"required"`
		}

		err := jsoniter.Unmarshal(in.RawPayload(), &data)
		if err == nil {
			err = exts.ValidateStruct(data)
		}
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: request.ClientId,
				Body: nex.WebSocketPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable parse payload: %v", err),
				}.Marshal(),
			})
			break
		}

		err = services.SetTypingStatus(data.ChannelID, uint(request.GetUserId()))
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: request.ClientId,
				Body: nex.WebSocketPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable boardcast status: %v", err),
				}.Marshal(),
			})
			break
		}
	case "events.subscribe", "events.unsubscribe", "events.unsubscribeAll":
		var data struct {
			ChannelID uint `json:"channel_id" validate:"required"`
		}

		err := jsoniter.Unmarshal(in.RawPayload(), &data)
		if err == nil {
			err = exts.ValidateStruct(data)
		}
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: request.ClientId,
				Body: nex.WebSocketPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable parse payload: %v", err),
				}.Marshal(),
			})
			break
		}

		action := strings.Split(in.Action, ".")[1]
		switch action {
		case "subscribe":
			services.SubscribeChannel(uint(request.GetUserId()), data.ChannelID, request.GetClientId())
		case "unsubscribe":
			services.UnsubscribeChannel(uint(request.GetUserId()), data.ChannelID)
		case "unsubscribeAll":
			services.UnsubscribeAll(uint(request.GetUserId()))
		}
	case "events.read":
		var data struct {
			ChannelMemberID uint `json:"channel_member_id" validate:"required"`
			EventID         uint `json:"event_id" validate:"required"`
		}

		err := jsoniter.Unmarshal(in.RawPayload(), &data)
		if err == nil {
			err = exts.ValidateStruct(data)
		}
		if err != nil {
			_, _ = sc.PushStream(context.Background(), &proto.PushStreamRequest{
				ClientId: request.ClientId,
				Body: nex.WebSocketPackage{
					Action:  "error",
					Message: fmt.Sprintf("unable parse payload: %v", err),
				}.Marshal(),
			})
			break
		}

		// WARN We trust the user here, so we don't need to check if the channel member is valid for performance
		services.SetReadingAnchor(data.ChannelMemberID, data.EventID)
	}

	return &proto.PushStreamResponse{}, nil
}
