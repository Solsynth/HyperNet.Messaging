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

	pivot := c.QueryInt("pivot", 0)
	if pivot < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "pivot must be greater than zero")
	}

	take := c.QueryInt("take", 10)
	if take > 100 {
		take = 100
	}

	tx := database.C

	var lookupRange []uint
	var ignoreRange []uint
	var channelMembers []models.ChannelMember
	if err := database.C.Where("account_id = ?", user.ID).Select("id", "channel_id").Find(&channelMembers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("unable to get channel identity of you: %v", err))
	} else {
		for _, member := range channelMembers {
			lookupRange = append(lookupRange, member.ChannelID)
			ignoreRange = append(ignoreRange, member.ID)
		}
	}

	tx = tx.Where("channel_id IN ?", lookupRange)
	tx = tx.Where("sender_id NOT IN ?", ignoreRange)
	tx = tx.Where("id > ?", pivot)

	countTx := tx
	var count int64
	if err := countTx.Model(&models.Event{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var items []models.Event
	if err := tx.
		Limit(take).
		Order("created_at DESC").
		Preload("Sender").
		Preload("Channel").
		Preload("Channel.Realm").
		Find(&items).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  items,
	})

}
