package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
)

func getWhatsNew(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var lookupRange []uint
	var lookupPivots []int
	var ignoreRange []uint
	var channelMembers []models.ChannelMember
	if err := database.C.Where("account_id = ?", user.ID).
		Select("id", "channel_id", "reading_anchor").
		Find(&channelMembers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("unable to get channel identity of you: %v", err))
	} else {
		for _, member := range channelMembers {
			if member.ReadingAnchor == nil {
				continue
			}
			lookupRange = append(lookupRange, member.ChannelID)
			lookupPivots = append(lookupPivots, *member.ReadingAnchor)
			ignoreRange = append(ignoreRange, member.ID)
		}
	}

	tx := database.C
	tx = tx.Where("channel_id IN ?", lookupRange)
	tx = tx.Where("sender_id NOT IN ?", ignoreRange)

	countTx := tx
	var count int64
	if err := countTx.Model(&models.Event{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var result []struct {
		ChannelID          uint `json:"channel_id"`
		UnreadMessageCount int  `json:"count"`
	}
	if err := database.C.Table("channel_members cm").
		Select("cm.channel_id, COUNT(m.id) AS unread_message_count").
		Joins("JOIN messages m ON m.channel_id = cm.channel_id").
		Where("m.id > cm.reading_anchor AND cm.account_id = ?", 1).
		Group("cm.channel_id").
		Scan(&result).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(result)
}
