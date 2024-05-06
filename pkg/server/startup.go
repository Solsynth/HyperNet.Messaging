package server

import (
	"net/http"
	"strings"
	"time"

	"git.solsynth.dev/hydrogen/messaging/pkg"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/favicon"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var A *fiber.App

func NewServer() {
	templates := html.NewFileSystem(http.FS(pkg.FS), ".gohtml")

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
		Views:                 templates,
		ViewsLayout:           "views/index",
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

		channels := api.Group("/channels/:realm").Use(realmMiddleware).Name("Channels API")
		{
			channels.Get("/", listChannel)
			channels.Get("/:channel", getChannel)
			channels.Get("/:channel/available", authMiddleware, getChannelAvailability)
			channels.Get("/me", authMiddleware, listOwnedChannel)
			channels.Get("/me/available", authMiddleware, listAvailableChannel)

			channels.Post("/", authMiddleware, createChannel)
			channels.Put("/:channelId", authMiddleware, editChannel)
			channels.Delete("/:channelId", authMiddleware, deleteChannel)

			channels.Get("/:channel/members", listChannelMembers)
			channels.Put("/:channel/members", authMiddleware, editChannelMembership)
			channels.Post("/:channel/members", authMiddleware, addChannelMember)
			channels.Post("/:channel/members/me", authMiddleware, joinChannel)
			channels.Delete("/:channel/members", authMiddleware, removeChannelMember)
			channels.Delete("/:channel/members/me", authMiddleware, leaveChannel)

			channels.Get("/:channel/messages", authMiddleware, listMessage)
			channels.Post("/:channel/messages", authMiddleware, newMessage)
			channels.Put("/:channel/messages/:messageId", authMiddleware, editMessage)
			channels.Delete("/:channel/messages/:messageId", authMiddleware, deleteMessage)

			channels.Get("/:channel/calls", listCall)
			channels.Get("/:channel/calls/ongoing", getOngoingCall)
			channels.Post("/:channel/calls", authMiddleware, startCall)
			channels.Delete("/:channel/calls/ongoing", authMiddleware, endCall)
			channels.Post("/:channel/calls/ongoing/token", authMiddleware, exchangeCallToken)
		}

		api.Get("/ws", authMiddleware, websocket.New(messageGateway))
	}

	A.Use(favicon.New(favicon.Config{
		FileSystem: http.FS(pkg.FS),
		File:       "views/favicon.png",
		URL:        "/favicon.png",
	}))

	A.Get("/", func(c *fiber.Ctx) error {
		return c.Render("views/open", fiber.Map{
			"frontend": viper.GetString("frontend"),
		})
	})
}

func Listen() {
	if err := A.Listen(viper.GetString("bind")); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when starting server...")
	}
}
