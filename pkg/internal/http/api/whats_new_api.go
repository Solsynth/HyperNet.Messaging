package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"github.com/gofiber/fiber/v2"
)

func getWhatsNew(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var result []struct {
		ChannelID          uint `json:"channel_id"`
		UnreadMessageCount int  `json:"count"`
	}
	if err := database.C.Table("channel_members cm").
		Select("cm.channel_id, COUNT(m.id) AS unread_message_count").
		Joins("JOIN events m ON m.channel_id = cm.channel_id").
		Where("m.id > cm.reading_anchor AND cm.account_id = ?", user.ID).
		Group("cm.channel_id").
		Scan(&result).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(result)
}
