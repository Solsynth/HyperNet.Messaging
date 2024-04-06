package services

import (
	"errors"
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/golang-jwt/jwt/v5"
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
		Provider:   models.CallProviderJitsi,
		ExternalID: channel.Name,
		FounderID:  founder.ID,
		ChannelID:  channel.ID,
		Founder:    founder,
		Channel:    channel,
	}

	if _, err := GetOngoingCall(channel); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return call, fmt.Errorf("this channel already has an ongoing call")
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

func EncodeCallToken(user models.Account) (string, error) {
	// Jitsi requires HS256 as algorithm, so we cannot use HS512
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"context": jwt.MapClaims{
			"user": jwt.MapClaims{
				"avatar": user.Avatar,
				"name":   user.Name,
			},
		},
		"aud":  viper.GetString("meeting.client_id"),
		"iss":  viper.GetString("domain"),
		"sub":  "meet.jitsi",
		"room": "*",
	})

	return tk.SignedString([]byte(viper.GetString("meeting.client_secret")))
}
