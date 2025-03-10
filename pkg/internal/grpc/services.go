package grpc

import (
	"context"

	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/nexus/pkg/proto"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
)

func (v *Server) BroadcastEvent(ctx context.Context, in *proto.EventInfo) (*proto.EventResponse, error) {
	log.Debug().Str("event", in.GetEvent()).
		Msg("Got a broadcasting event...")

	switch in.GetEvent() {
	// Clear the subscribed channel
	case "ws.client.unregister":
		// Update user last seen at
		data := nex.DecodeMap(in.GetData())
		id := data["id"].(string)
		services.UnsubscribeAllWithClient(id)
		log.Info().Str("client", id).Msg("Client unregistered, cleaning up subscribed channels...")
	// Account recycle
	case "deletion":
		data := nex.DecodeMap(in.GetData())
		resType, ok := data["type"].(string)
		if !ok {
			break
		}
		switch resType {
		case "account":
			var data struct {
				ID int `json:"id"`
			}
			if err := jsoniter.Unmarshal(in.GetData(), &data); err != nil {
				break
			}
			tx := database.C.Begin()
			for _, model := range database.AutoMaintainRange {
				switch model.(type) {
				default:
					tx.Delete(model, "account_id = ?", data.ID)
				}
			}
			tx.Commit()
		case "realm":
			var data struct {
				ID int `json:"id"`
			}
			if err := jsoniter.Unmarshal(in.GetData(), &data); err != nil {
				break
			}
			var channels []models.Channel
			if err := database.C.Where("realm_id = ?", data.ID).Find(&channels).Error; err != nil {
				break
			}
			for _, channel := range channels {
				_ = services.DeleteChannel(channel)
			}
		}
	}

	return &proto.EventResponse{}, nil
}
