package server

import (
	"net/http"
	"strings"
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg/views"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var A *fiber.App

func NewServer() {
	A = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		ServerHeader:          "Hydrogen.Messaging",
		AppName:               "Hydrogen.Messaging",
		ProxyHeader:           fiber.HeaderXForwardedFor,
		JSONEncoder:           jsoniter.ConfigCompatibleWithStandardLibrary.Marshal,
		JSONDecoder:           jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal,
		BodyLimit:             50 * 1024 * 1024,
		EnablePrintRoutes:     viper.GetBool("debug.print_routes"),
	})

	A.Use(idempotency.New())
	A.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodHead,
			fiber.MethodOptions,
			fiber.MethodPut,
			fiber.MethodDelete,
			fiber.MethodPatch,
		}, ","),
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))

	A.Use(logger.New(logger.Config{
		Format: "${status} | ${latency} | ${method} ${path}\n",
		Output: log.Logger,
	}))

	A.Get("/.well-known", getMetadata)

	api := A.Group("/api").Name("API")
	{
		api.Get("/users/me", authMiddleware, getUserinfo)
		api.Get("/users/:accountId", getOthersInfo)

		api.Get("/attachments/o/:fileId", cache.New(cache.Config{
			Expiration:   365 * 24 * time.Hour,
			CacheControl: true,
		}), readAttachment)
		api.Post("/attachments", authMiddleware, uploadAttachment)
		api.Delete("/attachments/:id", authMiddleware, deleteAttachment)

		channels := api.Group("/channels").Name("Channels API")
		{
			channels.Get("/", listChannel)
			channels.Get("/me", authMiddleware, listOwnedChannel)
			channels.Get("/me/available", authMiddleware, listAvailableChannel)
			channels.Get("/:channelId", getChannel)
			channels.Get("/:channelId/members", listChannelMembers)
			channels.Post("/", authMiddleware, createChannel)
			channels.Post("/:channelId/invite", authMiddleware, inviteChannel)
			channels.Post("/:channelId/kick", authMiddleware, kickChannel)
			channels.Put("/:channelId", authMiddleware, editChannel)
			channels.Delete("/:channelId", authMiddleware, deleteChannel)
		}
	}

	A.Use("/", cache.New(cache.Config{
		Expiration:   24 * time.Hour,
		CacheControl: true,
	}), filesystem.New(filesystem.Config{
		Root:         http.FS(views.FS),
		PathPrefix:   "dist",
		Index:        "index.html",
		NotFoundFile: "dist/index.html",
	}))
}

func Listen() {
	if err := A.Listen(viper.GetString("bind")); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when starting server...")
	}
}
