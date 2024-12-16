package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/nex"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"git.solsynth.dev/hypernet/pusher/pkg/pushkit"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func ListCall(channel models.Channel, take, offset int) ([]models.Call, error) {
	var calls []models.Call
	if err := database.C.
		Where(models.Call{ChannelID: channel.ID}).
		Limit(take).
		Offset(offset).
		Preload("Founder").
		Preload("Channel").
		Order("created_at DESC").
		Find(&calls).Error; err != nil {
		return calls, err
	} else {
		return calls, nil
	}
}

func GetCall(channel models.Channel, id uint) (models.Call, error) {
	var call models.Call
	if err := database.C.
		Where(models.Call{
			BaseModel: cruda.BaseModel{ID: id},
			ChannelID: channel.ID,
		}).
		Preload("Founder").
		Preload("Channel").
		Order("created_at DESC").
		First(&call).Error; err != nil {
		return call, err
	} else {
		return call, nil
	}
}

func GetOngoingCall(channel models.Channel) (models.Call, error) {
	var call models.Call
	if err := database.C.
		Where(models.Call{ChannelID: channel.ID}).
		Where("ended_at IS NULL").
		Preload("Founder").
		Preload("Channel").
		Order("created_at DESC").
		First(&call).Error; err != nil {
		return call, err
	} else {
		return call, nil
	}
}

func GetCallParticipants(call models.Call) ([]*livekit.ParticipantInfo, error) {
	res, err := Lk.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: call.ExternalID,
	})
	if err != nil {
		return nil, err
	}
	return res.Participants, nil
}

func NewCall(channel models.Channel, founder models.ChannelMember) (models.Call, error) {
	id := fmt.Sprintf("%s+%d", channel.Alias, channel.ID)
	call := models.Call{
		ExternalID: id,
		FounderID:  founder.ID,
		ChannelID:  channel.ID,
		Founder:    founder,
		Channel:    channel,
	}

	if _, err := GetOngoingCall(channel); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return call, fmt.Errorf("this channel already has an ongoing call")
	}

	_, err := Lk.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            id,
		EmptyTimeout:    viper.GetUint32("calling.empty_timeout_duration"),
		MaxParticipants: viper.GetUint32("calling.max_participants"),
	})
	if err != nil {
		return call, fmt.Errorf("remote livekit error: %v", err)
	}

	var members []models.ChannelMember
	if err := database.C.Save(&call).Error; err != nil {
		return call, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: call.ChannelID,
	}).Find(&members).Error; err == nil {
		call, _ = GetCall(call.Channel, call.ID)
		var pendingUsers []uint64
		for _, member := range members {
			if member.ID != call.Founder.ID {
				pendingUsers = append(pendingUsers, uint64(member.AccountID))
			}
			PushCommand(member.AccountID, nex.WebSocketPackage{
				Action:  "calls.new",
				Payload: call,
			})
		}

		channel, _ = GetChannel(channel.ID)
		if channel.RealmID != nil {
			realm, err := authkit.GetRealm(gap.Nx, *channel.RealmID)
			if err == nil {
				channel.Realm = &realm
			}
		}

		err = authkit.NotifyUserBatch(
			gap.Nx,
			pendingUsers,
			pushkit.Notification{
				Topic: "messaging.callStart",
				Title: fmt.Sprintf("Call in (%s)", channel.DisplayText()),
				Body:  fmt.Sprintf("%s is calling", call.Founder.Name),
				Metadata: map[string]any{
					"avatar":     call.Founder.Avatar,
					"user_id":    call.Founder.AccountID,
					"user_name":  call.Founder.Name,
					"user_nick":  call.Founder.Nick,
					"channel_id": call.ChannelID,
				},
				Priority: 5,
			},
		)
		if err != nil {
			log.Warn().Err(err).Msg("An error occurred when trying notify user.")
		}
	}

	return call, nil
}

func EndCall(call models.Call) (models.Call, error) {
	call.EndedAt = lo.ToPtr(time.Now())

	if _, err := Lk.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: call.ExternalID,
	}); err != nil {
		log.Error().Err(err).Msg("Unable to delete room at livekit side")
	}

	var members []models.ChannelMember
	if err := database.C.Save(&call).Error; err != nil {
		return call, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: call.ChannelID,
	}).Find(&members).Error; err == nil {
		call, _ = GetCall(call.Channel, call.ID)
		for _, member := range members {
			PushCommand(member.AccountID, nex.WebSocketPackage{
				Action:  "calls.end",
				Payload: call,
			})
		}
	}

	return call, nil
}

func KickParticipantInCall(call models.Call, username string) error {
	_, err := Lk.RemoveParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     call.ExternalID,
		Identity: username,
	})
	return err
}

func EncodeCallToken(user authm.Account, call models.Call) (string, error) {
	isAdmin := user.ID == call.FounderID || user.ID == call.Channel.AccountID

	grant := &auth.VideoGrant{
		Room:      call.ExternalID,
		RoomJoin:  true,
		RoomAdmin: isAdmin,
	}

	metadata, _ := jsoniter.Marshal(user)

	duration := time.Second * time.Duration(viper.GetInt("calling.token_duration"))
	tk := auth.NewAccessToken(viper.GetString("calling.api_key"), viper.GetString("calling.api_secret"))
	tk.AddGrant(grant).
		SetIdentity(user.Name).
		SetName(user.Nick).
		SetMetadata(string(metadata)).
		SetValidFor(duration)

	return tk.ToJWT()
}
