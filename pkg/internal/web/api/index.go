package api

import (
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		quick := api.Group("/quick")
		{
			quick.Post("/:channelId/reply/:eventId", quickReply)
		}

		api.Get("/channels/me", listOwnedChannelGlobalWide)
		api.Get("/channels/me/available", listAvailableChannelGlobalWide)

		channels := api.Group("/channels/:realm").Use(realmMiddleware).Name("Channels API")
		{
			channels.Get("/", listChannel)
			channels.Get("/public", listPublicChannel)
			channels.Get("/me", listOwnedChannel)
			channels.Get("/me/available", listAvailableChannel)
			channels.Get("/:channel", getChannel)
			channels.Get("/:channel/me", getChannelIdentity)
			channels.Get("/:channel/members/me", getChannelIdentity)
			channels.Put("/:channel/me", editChannelIdentity)
			channels.Put("/:channel/me/notify", editChannelNotifyLevel)
			channels.Put("/:channel/members/me/notify", editChannelNotifyLevel)
			channels.Delete("/:channel/me", deleteChannelIdentity)

			channels.Post("/", createChannel)
			channels.Post("/dm", createDirectChannel)
			channels.Put("/:channelId", editChannel)
			channels.Delete("/:channelId", deleteChannel)
			channels.Get("/:channel/members", listChannelMembers)
			channels.Post("/:channel/members", addChannelMember)
			channels.Delete("/:channel/members/:memberId", removeChannelMember)

			channels.Get("/:channel/events", listEvent)
			channels.Get("/:channel/events/update", checkHasNewEvent)
			channels.Get("/:channel/events/:eventId", getEvent)
			channels.Post("/:channel/events", newRawEvent)

			channels.Post("/:channel/messages", newMessageEvent)
			channels.Put("/:channel/messages/:messageId", editMessageEvent)
			channels.Delete("/:channel/messages/:messageId", deleteMessageEvent)

			channels.Get("/:channel/calls", listCall)
			channels.Get("/:channel/calls/ongoing", getOngoingCall)
			channels.Post("/:channel/calls", startCall)
			channels.Delete("/:channel/calls/ongoing", endCall)
			channels.Delete("/:channel/calls/ongoing/participant", kickParticipantInCall)
			channels.Post("/:channel/calls/ongoing/token", exchangeCallToken)
		}

		api.Get("/whats-new", getWhatsNew)
	}
}
