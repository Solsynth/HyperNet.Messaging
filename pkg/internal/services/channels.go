package services

import (
	"fmt"
	"regexp"
	"time"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/cachekit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type channelIdentityCacheEntry struct {
	Channel       models.Channel       `json:"channel"`
	ChannelMember models.ChannelMember `json:"channel_member"`
}

func KgChannelIdentityCache(channel string, user uint, realm ...uint) string {
	if len(realm) > 0 {
		return fmt.Sprintf("channel-identity-%s#%d@%d", channel, user, realm)
	} else {
		return fmt.Sprintf("channel-identity-%s#%d", channel, user)
	}
}

func CacheChannelIdentity(channel models.Channel, member models.ChannelMember, user uint, realm ...uint) {
	key := KgChannelIdentityCache(channel.Alias, user, realm...)

	cachekit.Set(
		gap.Ca,
		key,
		channelIdentityCacheEntry{channel, member},
		60*time.Minute,
		fmt.Sprintf("channel#%d", channel.ID),
		fmt.Sprintf("user#%d", user),
	)
}

func GetChannelIdentityWithID(id uint, user uint) (models.Channel, models.ChannelMember, error) {
	var member models.ChannelMember

	if err := database.C.Where(models.ChannelMember{
		AccountID: user,
		ChannelID: id,
	}).Preload("Channel").First(&member).Error; err != nil {
		return member.Channel, member, fmt.Errorf("channel principal not found: %v", err.Error())
	}

	return member.Channel, member, nil
}

func GetChannelIdentity(alias string, user uint, realm ...authm.Realm) (models.Channel, models.ChannelMember, error) {
	var err error
	var channel models.Channel
	var member models.ChannelMember

	hitCache := false
	if len(realm) > 0 {
		if val, err := cachekit.Get[channelIdentityCacheEntry](
			gap.Ca,
			KgChannelIdentityCache(alias, user, realm[0].ID),
		); err == nil {
			channel = val.Channel
			member = val.ChannelMember
			hitCache = true
		}
	} else {
		if val, err := cachekit.Get[channelIdentityCacheEntry](
			gap.Ca,
			KgChannelIdentityCache(alias, user),
		); err == nil {
			channel = val.Channel
			member = val.ChannelMember
			hitCache = true
		}
	}
	if !hitCache {
		if len(realm) > 0 {
			channel, member, err = GetAvailableChannelWithAlias(alias, user, realm[0].ID)
			CacheChannelIdentity(channel, member, user, realm[0].ID)
		} else {
			channel, member, err = GetAvailableChannelWithAlias(alias, user)
			CacheChannelIdentity(channel, member, user)
		}
	}

	return channel, member, err
}

func GetChannelAliasAvailability(alias string) error {
	if !regexp.MustCompile("^[a-z0-9-]+$").MatchString(alias) {
		return fmt.Errorf("channel alias should only contains lowercase letters, numbers, and hyphens")
	}
	return nil
}

func GetChannel(id uint) (models.Channel, error) {
	var channel models.Channel
	tx := database.C.Where("id = ?", id)
	tx = PreloadDirectChannelMembers(tx)
	if err := tx.First(&channel).Error; err != nil {
		return channel, err
	}

	return channel, nil
}

func GetChannelWithAlias(alias string, realmId ...uint) (models.Channel, error) {
	var channel models.Channel
	tx := database.C.Where(models.Channel{Alias: alias})
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	} else {
		tx = tx.Where("realm_id IS NULL")
	}
	tx = PreloadDirectChannelMembers(tx)
	if err := tx.First(&channel).Error; err != nil {
		return channel, err
	}

	return channel, nil
}

func GetAvailableChannelWithAlias(alias string, user uint, realmId ...uint) (models.Channel, models.ChannelMember, error) {
	var err error
	var member models.ChannelMember
	var channel models.Channel
	if channel, err = GetChannelWithAlias(alias, realmId...); err != nil {
		return channel, member, err
	}

	if err := database.C.Where(models.ChannelMember{
		AccountID: user,
		ChannelID: channel.ID,
	}).First(&member).Error; err != nil {
		return channel, member, fmt.Errorf("channel principal not found: %v", err.Error())
	}

	return channel, member, nil
}

func GetAvailableChannel(id uint, user authm.Account) (models.Channel, models.ChannelMember, error) {
	var err error
	var member models.ChannelMember
	var channel models.Channel
	if channel, err = GetChannel(id); err != nil {
		return channel, member, err
	}
	tx := database.C.Where(models.ChannelMember{
		AccountID: user.ID,
		ChannelID: channel.ID,
	})
	if err := tx.First(&member).Error; err != nil {
		return channel, member, fmt.Errorf("channel principal not found: %v", err.Error())
	}

	return channel, member, nil
}

func PreloadDirectChannelMembers(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Members", func(db *gorm.DB) *gorm.DB {
		return db.Joins(
			fmt.Sprintf(
				"JOIN %schannels AS c ON c.type = ?",
				viper.GetString("database.prefix"),
			),
			models.ChannelTypeDirect,
		)
	})
}

func ListChannel(user *authm.Account, realmId ...uint) ([]models.Channel, error) {
	var identities []models.ChannelMember
	var idRange []uint
	if user != nil {
		if err := database.C.Where("account_id = ?", user.ID).Find(&identities).Error; err != nil {
			return nil, fmt.Errorf("unable to get identities: %v", err)
		}
		for _, identity := range identities {
			idRange = append(idRange, identity.ChannelID)
		}
	}

	var channels []models.Channel
	tx := database.C
	tx = tx.Where("id IN ? OR is_public = true", idRange)
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListChannelPublic(realmId ...uint) ([]models.Channel, error) {
	var channels []models.Channel
	tx := database.C
	tx = tx.Where("is_public = true")
	if len(realmId) > 0 {
		tx = tx.Where("realm_id = ?", realmId)
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListChannelWithUser(user authm.Account, realmId ...uint) ([]models.Channel, error) {
	var channels []models.Channel
	tx := database.C.Where(&models.Channel{AccountID: user.ID})
	if len(realmId) > 0 {
		if realmId[0] != 0 {
			tx = tx.Where("realm_id = ?", realmId)
		}
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func ListAvailableChannel(tx *gorm.DB, user authm.Account, realmId ...uint) ([]models.Channel, error) {
	var channels []models.Channel
	var members []models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		AccountID: user.ID,
	}).Find(&members).Error; err != nil {
		return channels, err
	}

	idx := lo.Map(members, func(item models.ChannelMember, index int) uint {
		return item.ChannelID
	})

	tx = tx.Where("id IN ?", idx)
	if len(realmId) > 0 {
		if realmId[0] != 0 {
			tx = tx.Where("realm_id = ?", realmId)
		}
	} else {
		tx = tx.Where("realm_id IS NULL")
	}

	tx = PreloadDirectChannelMembers(tx)

	if err := tx.Find(&channels).Error; err != nil {
		return channels, err
	}

	return channels, nil
}

func NewChannel(channel models.Channel) (models.Channel, error) {
	err := database.C.Save(&channel).Error
	return channel, err
}

func EditChannel(channel models.Channel) (models.Channel, error) {
	err := database.C.Save(&channel).Error

	if err == nil {
		cachekit.DeleteByTags(gap.Ca, fmt.Sprintf("channel#%d", channel.ID))
	}

	return channel, err
}

func DeleteChannel(channel models.Channel) error {
	if err := database.C.Delete(&channel).Error; err == nil {
		UnsubscribeAllWithChannels(channel.ID)

		database.C.Where("channel_id = ?", channel.ID).Delete(&models.Event{})
		database.C.Where("channel_id = ?", channel.ID).Delete(&models.ChannelMember{})

		cachekit.DeleteByTags(gap.Ca, fmt.Sprintf("channel#%d", channel.ID))

		return nil
	} else {
		return err
	}
}
