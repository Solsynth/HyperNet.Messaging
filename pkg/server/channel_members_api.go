package server

import (
	"fmt"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/fiber/v2"
)

func listChannelMembers(c *fiber.Ctx) error {
	alias := c.Params("channel")

	channel, err := services.GetChannelWithAlias(alias)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if members, err := services.ListChannelMember(channel.ID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(members)
	}
}

func addChannelMember(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var data struct {
		AccountName string `json:"account_name" validate:"required"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias:     alias,
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var account models.Account
	if err := database.C.Where(&models.Account{
		Name: data.AccountName,
	}).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.AddChannelMemberWithCheck(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func removeChannelMember(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var data struct {
		AccountName string `json:"account_name" validate:"required"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias:     alias,
		AccountID: user.ID,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var account models.Account
	if err := database.C.Where(&models.Account{
		Name: data.AccountName,
	}).First(&account).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if err := services.RemoveChannelMember(account, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func editChannelMembership(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var data struct {
		NotifyLevel int8 `json:"notify_level"`
	}

	if err := BindAndValidate(c, &data); err != nil {
		return err
	}

	channel, err := services.GetChannelWithAlias(alias)
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
	user := c.Locals("principal").(models.Account)
	alias := c.Params("channel")

	var channel models.Channel
	if err := database.C.Where(&models.Channel{
		Alias: alias,
	}).First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if _, _, err := services.GetAvailableChannel(channel.ID, user); err == nil {
		return fiber.NewError(fiber.StatusBadRequest, "you already joined the channel")
	} else if channel.RealmID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "you was impossible to join a channel without related realm")
	}

	if realm, err := services.GetRealm(channel.ID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("invalid channel, related realm was not found: %v", err))
	} else if _, err := services.GetRealmMember(realm.ID, user.ExternalID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("you are not a part of the realm: %v", err))
	}

	if err := services.AddChannelMember(user, channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}

func leaveChannel(c *fiber.Ctx) error {
	user := c.Locals("principal").(models.Account)
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
