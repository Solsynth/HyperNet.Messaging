package api

import (
	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
)

func getUserinfo(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data models.Account
	if err := database.C.
		Where(&hyper.BaseModel{ID: user.ID}).
		First(&data).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(data)
}

func getOthersInfo(c *fiber.Ctx) error {
	accountId := c.Params("accountId")

	var data models.Account
	if err := database.C.
		Where(&hyper.BaseUser{Name: accountId}).
		First(&data).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(data)
}
