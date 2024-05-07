package server

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/gofiber/contrib/websocket"
	jsoniter "github.com/json-iterator/go"
)

func messageGateway(c *websocket.Conn) {
	user := c.Locals("principal").(models.Account)

	// Push connection
	services.ClientRegister(user, c)

	// Event loop
	var task models.UnifiedCommand

	var messageType int
	var packet []byte
	var err error

	for {
		if messageType, packet, err = c.ReadMessage(); err != nil {
			break
		} else if err := jsoniter.Unmarshal(packet, &task); err != nil {
			_ = c.WriteMessage(messageType, models.UnifiedCommand{
				Action:  "error",
				Message: "unable to unmarshal your command, requires json request",
			}.Marshal())
			continue
		}

		message := services.DealCommand(task, user)

		if message != nil {
			if err = c.WriteMessage(messageType, message.Marshal()); err != nil {
				break
			}
		}
	}

	// Pop connection
	services.ClientUnregister(user, c)
}
