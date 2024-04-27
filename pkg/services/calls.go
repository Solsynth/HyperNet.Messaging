package services

import (
	"context"
	"errors"
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/external"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"time"
)

func ListCall(channel models.Channel, take, offset int) ([]models.Call, error) {
	var calls []models.Call
	if err := database.C.
		Where(models.Call{ChannelID: channel.ID}).
		Limit(take).
		Offset(offset).
		Preload("Founder").
		Preload("Founder.Account").
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
			BaseModel: models.BaseModel{ID: id},
			ChannelID: channel.ID,
		}).
		Preload("Founder").
		Preload("Founder.Account").
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

func NewCall(channel models.Channel, founder models.ChannelMember) (models.Call, error) {
	call := models.Call{
		ExternalID: channel.Alias,
		FounderID:  founder.ID,
		ChannelID:  channel.ID,
		Founder:    founder,
		Channel:    channel,
	}

	if _, err := GetOngoingCall(channel); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return call, fmt.Errorf("this channel already has an ongoing call")
	}

	_, err := external.Lk.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            call.ExternalID,
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
	}).Preload("Account").Find(&members).Error; err == nil {
		channel := call.Channel
		call, _ = GetCall(call.Channel, call.ID)
		for _, member := range members {
			if member.ID != call.Founder.ID {
				if member.Notify == models.NotifyLevelAll {
					err = NotifyAccount(member.Account,
						fmt.Sprintf("New Call #%s", channel.Alias),
						fmt.Sprintf("%s starts a new call", call.Founder.Account.Name),
						false,
					)
					if err != nil {
						log.Warn().Err(err).Msg("An error occurred when trying notify user.")
					}
				}
			}
			PushCommand(member.AccountID, models.UnifiedCommand{
				Action:  "calls.new",
				Payload: call,
			})
		}
	}

	return call, nil
}

func EndCall(call models.Call) (models.Call, error) {
	call.EndedAt = lo.ToPtr(time.Now())

	if _, err := external.Lk.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: call.ExternalID,
	}); err != nil {
		log.Error().Err(err).Msg("Unable to delete room at livekit side")
	}

	var members []models.ChannelMember
	if err := database.C.Save(&call).Error; err != nil {
		return call, err
	} else if err = database.C.Where(models.ChannelMember{
		ChannelID: call.ChannelID,
	}).Preload("Account").Find(&members).Error; err == nil {
		call, _ = GetCall(call.Channel, call.ID)
		for _, member := range members {
			PushCommand(member.AccountID, models.UnifiedCommand{
				Action:  "calls.end",
				Payload: call,
			})
		}
	}

	return call, nil
}

func EncodeCallToken(user models.Account, call models.Call) (string, error) {
	isAdmin := false
	if user.ID == call.FounderID || user.ID == call.Channel.AccountID {
		isAdmin = true
	}

	grant := &auth.VideoGrant{
		Room:      call.ExternalID,
		RoomJoin:  true,
		RoomAdmin: isAdmin,
	}

	duration := time.Second * time.Duration(viper.GetInt("calling.token_duration"))
	tk := auth.NewAccessToken(viper.GetString("calling.api_key"), viper.GetString("calling.api_secret"))
	tk.AddGrant(grant).SetIdentity(user.Name).SetValidFor(duration)

	return tk.ToJWT()
}
