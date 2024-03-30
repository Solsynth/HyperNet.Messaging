package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/gofiber/contrib/websocket"
	"github.com/samber/lo"
)

var WsConn = make(map[uint][]*websocket.Conn)

func CheckOnline(user models.Account) bool {
	return len(WsConn[user.ID]) > 0
}

func PushCommand(userId uint, task models.UnifiedCommand) {
	for _, conn := range WsConn[userId] {
		_ = conn.WriteMessage(1, task.Marshal())
	}
}

func DealCommand(task models.UnifiedCommand, user models.Account) *models.UnifiedCommand {
	switch task.Action {
	case "messages.send.text":
		var req struct {
			ChannelID uint   `json:"channel_id"`
			Content   string `json:"content"`
		}
		models.FitStruct(task.Payload, &req)

		if channel, member, err := GetAvailableChannel(req.ChannelID, user); err != nil {
			return lo.ToPtr(models.UnifiedCommandFromError(err))
		} else if _, err = NewTextMessage(req.Content, member, channel); err != nil {
			return lo.ToPtr(models.UnifiedCommandFromError(err))
		} else {
			return nil
		}
	default:
		return &models.UnifiedCommand{
			Action:  "error",
			Message: "command not found",
		}
	}
}
