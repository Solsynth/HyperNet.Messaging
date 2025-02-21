package api

import (
	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/http/exts"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	authm "git.solsynth.dev/hypernet/passport/pkg/authkit/models"
	"strconv"

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

func addChannelMember(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

	var data struct {
		Related string `json:"related" validate:"required"`
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

	if !channel.IsPublic {
		if member, err := services.GetChannelMember(user, channel.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		} else if member.PowerLevel < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of a channel to add member into it")
		}
	}

	var err error
	var account authm.Account
	var numericId int
	if numericId, err = strconv.Atoi(data.Related); err == nil {
		account, err = authkit.GetUser(gap.Nx, uint(numericId))
	} else {
		account, err = authkit.GetUserByName(gap.Nx, data.Related)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.AddChannelMemberWithCheck(account, user, channel); err != nil {
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
	memberId := c.Params("memberId")

	numericId, err := strconv.Atoi(memberId)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if channel.Type == models.ChannelTypeDirect {
		return fiber.NewError(fiber.StatusBadRequest, "direct message member changes was not allowed")
	} else if channel.AccountID == user.ID {
		return fiber.NewError(fiber.StatusBadRequest, "you cannot remove yourself from your own channel")
	}

	var member models.ChannelMember
	if me, err := services.GetChannelMember(user, channel.ID); err != nil {
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	} else if me.PowerLevel < 50 {
		return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of a channel to remove member from it")
	}

	if err := database.C.Where(&models.ChannelMember{
		BaseModel: cruda.BaseModel{ID: uint(numericId)},
		ChannelID: channel.ID,
	}).First(&member).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.RemoveChannelMember(member, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func deleteChannelIdentity(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	alias := c.Params("channel")

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

	if err = services.RemoveChannelMember(membership, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(membership)
	}
}

func editChannelIdentity(c *fiber.Ctx) error {
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

func editChannelNotifyLevel(c *fiber.Ctx) error {
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
