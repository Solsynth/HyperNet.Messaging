package api

import (
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		api.Get("/users/me", getUserinfo)
		api.Get("/users/:accountId", getOthersInfo)

		channels := api.Group("/channels/:realm").Use(realmMiddleware).Name("Channels API")
		{
			channels.Get("/", listChannel)
			channels.Get("/me", listOwnedChannel)
			channels.Get("/me/available", listAvailableChannel)
			channels.Get("/:channel", getChannel)
			channels.Get("/:channel/me", getChannelIdentity)

			channels.Post("/", createChannel)
			channels.Post("/dm", createDirectChannel)
			channels.Put("/:channelId", editChannel)
			channels.Delete("/:channelId", deleteChannel)

			channels.Get("/:channel/members", listChannelMembers)
			channels.Get("/:channel/members/me", getMyChannelMembership)
			channels.Put("/:channel/members/me", editMyChannelMembership)
			channels.Post("/:channel/members", addChannelMember)
			channels.Post("/:channel/members/me", joinChannel)
			channels.Delete("/:channel/members", removeChannelMember)
			channels.Delete("/:channel/members/me", leaveChannel)

			channels.Get("/:channel/events", listEvent)
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
