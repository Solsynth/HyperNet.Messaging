package api

import (
	"fmt"

	"git.solsynth.dev/hydrogen/messaging/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func realmMiddleware(c *fiber.Ctx) error {
	realmAlias := c.Params("realm")
	if len(realmAlias) > 0 && realmAlias != "global" {
		realm, err := services.GetRealmWithAlias(realmAlias)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("requested channel with realm, but realm was not found: %v", err))
		} else {
			c.Locals("realm", realm)
		}
	}

	return c.Next()
}
