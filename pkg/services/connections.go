package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/models"
	"github.com/gofiber/contrib/websocket"
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
	default:
		return &models.UnifiedCommand{
			Action:  "error",
			Message: "command not found",
		}
	}
}
