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
	"github.com/samber/lo"
)

func getChannel(c *fiber.Ctx) error {
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

	return c.JSON(channel)
}

func getChannelIdentity(c *fiber.Ctx) error {
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

func listChannel(c *fiber.Ctx) error {
	var user *authm.Account
	if err := sec.EnsureAuthenticated(c); err == nil {
		user = lo.ToPtr(c.Locals("user").(authm.Account))
	}

	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channels, err = services.ListChannel(user, val.ID)
	} else {
		channels, err = services.ListChannel(user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listPublicChannel(c *fiber.Ctx) error {
	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channels, err = services.ListChannelPublic(val.ID)
	} else {
		channels, err = services.ListChannelPublic()
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listOwnedChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channels, err = services.ListChannelWithUser(user, val.ID)
	} else {
		channels, err = services.ListChannelWithUser(user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func listAvailableChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	tx := database.C
	isDirect := c.QueryBool("direct", false)
	if isDirect {
		tx = tx.Where("type = ?", models.ChannelTypeDirect)
	} else {
		tx = tx.Where("type = ?", models.ChannelTypeCommon)
	}

	var err error
	var channels []models.Channel
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		channels, err = services.ListAvailableChannel(tx, user, val.ID)
	} else {
		channels, err = services.ListAvailableChannel(tx, user)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channels)
}

func createChannel(c *fiber.Ctx) error {
	if err := sec.EnsureGrantedPerm(c, "CreateChannels", true); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)

	var data struct {
		Alias       string `json:"alias" validate:"required,lowercase,min=4,max=32"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
		IsCommunity bool   `json:"is_community"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	} else if err = services.GetChannelAliasAvailability(data.Alias); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var realm *authm.Realm
	if val, ok := c.Locals("realm").(authm.Realm); ok {
		if info, err := authkit.GetRealmMember(gap.Nx, val.ID, user.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can create channel related to it")
		} else if info.PowerLevel < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can create channel related to it")
		} else {
			realm = &val
		}
	}

	channel := models.Channel{
		Alias:       data.Alias,
		Name:        data.Name,
		Description: data.Description,
		AccountID:   user.ID,
		Type:        models.ChannelTypeCommon,
		IsPublic:    data.IsPublic,
		IsCommunity: data.IsCommunity,
		Members: []models.ChannelMember{
			{AccountID: user.ID, PowerLevel: 100},
		},
	}

	if realm != nil {
		channel.RealmID = &realm.ID
	}

	channel, err := services.NewChannel(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}

func editChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	id, _ := c.ParamsInt("channelId", 0)

	var data struct {
		Alias           string  `json:"alias" validate:"required,min=4,max=32"`
		Name            string  `json:"name" validate:"required"`
		Description     string  `json:"description"`
		IsPublic        bool    `json:"is_public"`
		IsCommunity     bool    `json:"is_community"`
		NewBelongsRealm *string `json:"new_belongs_realm"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	tx := database.C.Where("id = ?", id)

	if val, ok := c.Locals("realm").(authm.Realm); ok {
		if info, err := authkit.GetRealmMember(gap.Nx, val.ID, user.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can edit channel related to it")
		} else if info.PowerLevel < 50 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can edit channel related to it")
		} else {
			tx = tx.Where("realm_id = ?", val.ID)
		}
	} else {
		tx = tx.Where("realm_id IS NULL")
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if channel.RealmID != nil {
		if member, err := services.GetChannelMember(user, channel.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must be a part of this channel to edit it")
		} else if member.PowerLevel < 100 {
			return fiber.NewError(fiber.StatusForbidden, "you must be channel admin to edit it")
		}
	}

	if data.NewBelongsRealm != nil {
		if *data.NewBelongsRealm == "global" {
			channel.RealmID = nil
		} else {
			realm, err := authkit.GetRealmByAlias(gap.Nx, *data.NewBelongsRealm)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("requested channel with realm, but realm was not found: %v", err))
			} else {
				if info, err := authkit.GetRealmMember(gap.Nx, realm.ID, user.ID); err != nil {
					return fiber.NewError(fiber.StatusForbidden, "you must be a part of that realm then can transfer channel related to it")
				} else if info.PowerLevel < 50 {
					return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of that realm then can transfer channel related to it")
				} else {
					channel.RealmID = &realm.ID
				}
			}
		}
	}

	channel.Alias = data.Alias
	channel.Name = data.Name
	channel.Description = data.Description
	channel.IsPublic = data.IsPublic
	channel.IsCommunity = data.IsCommunity

	channel, err := services.EditChannel(channel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(channel)
}

func deleteChannel(c *fiber.Ctx) error {
	if err := sec.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(authm.Account)
	id, _ := c.ParamsInt("channelId", 0)

	tx := database.C.Where("id = ?", id)

	if val, ok := c.Locals("realm").(authm.Realm); ok {
		if info, err := authkit.GetRealmMember(gap.Nx, val.ID, user.ID); err != nil {
			return fmt.Errorf("you must be a part of that realm then can delete channel related to it")
		} else if info.PowerLevel < 50 {
			return fmt.Errorf("you must be a moderator of that realm then can delete channel related to it")
		} else {
			tx = tx.Where("realm_id = ?", val.ID)
		}
	} else {
		tx = tx.Where("(account_id = ? OR type = ?) AND realm_id IS NULL", user.ID, models.ChannelTypeDirect)
	}

	var channel models.Channel
	if err := tx.First(&channel).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if channel.Type == models.ChannelTypeDirect {
		if member, err := services.GetChannelMember(user, channel.ID); err != nil {
			return fiber.NewError(fiber.StatusForbidden, "you must related to this direct message if you want delete it")
		} else if member.PowerLevel < 100 {
			return fiber.NewError(fiber.StatusForbidden, "you must be a moderator of this direct message if you want delete it")
		}
	}

	if err := services.DeleteChannel(channel); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
