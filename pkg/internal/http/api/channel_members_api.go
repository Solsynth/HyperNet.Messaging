package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/http/exts"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func listChannelMembers(c *fiber.Ctx) error {
	alias := c.Params("channel")
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	count, err := services.CountChannelMember(channel.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if members, err := services.ListChannelMember(channel.ID, take, offset); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(fiber.Map{
			"count": count,
			"data":  members,
		})
	}
}

func getChannelProfileOfMyself(c *fiber.Ctx) error {
	alias := c.Params("channel")
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if member, err := services.GetChannelMember(user, channel.ID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else {
		return c.JSON(member)
	}
}

func addChannelMember(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Target string `json:"target" validate:"required"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if channel.Type == models.ChannelTypeDirect {
		return fiber.NewError(fiber.StatusBadRequest, "direct message member changes was not allowed")
	}

	if member, err := services.GetChannelMember(user, channel.ID); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	} else if member.PowerLevel < 50 {
		return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of a channel to add member into it")
	}

	var account authm.Account
	if err := database.C.Where("name = ?", data.Target).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.AddChannelMemberWithCheck(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func removeChannelMember(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Target string `json:"target" validate:"required"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias:     alias,
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if channel.Type == models.ChannelTypeDirect {
		return fiber.NewError(fiber.StatusBadRequest, "direct message member changes was not allowed")
	}

	if member, err := services.GetChannelMember(user, channel.ID); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	} else if member.PowerLevel < 50 {
		return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of a channel to remove member into it")
	}

	var account authm.Account
	if err := database.C.Where("name = ?", data.Target).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.RemoveChannelMember(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func editChannelProfileOfMyself(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Nick string `json:"nick"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var membership models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channel.ID,
		AccountID: user.ID,
	}).First(&membership).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	membership.Name = user.Name
	if len(data.Nick) > 0 {
		membership.Nick = data.Nick
	} else {
		membership.Nick = user.Nick
	}

	if membership, err := services.EditChannelMember(membership); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(membership)
	}
}

func editChannelNotifyLevelOfMyself(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		NotifyLevel int8 `json:"notify_level"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var err error
	var channel models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channel, err = services.GetChannelWithAlias(alias, val.ID)
	} else {
		channel, err = services.GetChannelWithAlias(alias)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var membership models.ChannelMember
	if err := database.C.Where(&models.ChannelMember{
		ChannelID: channel.ID,
		AccountID: user.ID,
	}).First(&membership).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	membership.Notify = data.NotifyLevel

	if membership, err := services.EditChannelMember(membership); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(membership)
	}
}

func joinChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err == nil {
		return fiber.NewError(fiber.StatusBadRequest, "you already joined the channel")
	} else if channel.RealmID == nil && !channel.IsCommunity {
		return fiber.NewError(fiber.StatusBadRequest, "you were impossible to join a channel without related realm and non-community")
	}

	if channel.RealmID != nil {
		if _, err := authkit.GetRealmMember(gap.Nx, *channel.RealmID, user.ID); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("you are not a part of the realm: %v", err))
		}
	}

	if err := services.AddChannelMember(user, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func leaveChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if user.ID == channel.AccountID {
		return fiber.NewError(fiber.StatusBadRequest, "you cannot leave your own channel")
	}

	if err := services.RemoveChannelMember(user, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}
