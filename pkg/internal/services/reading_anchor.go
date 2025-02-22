package services

import (
	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var readingAnchorQueue = make(map[uint]uint)

func SetReadingAnchor(memberId uint, eventId uint) {
	if val, ok := readingAnchorQueue[memberId]; ok {
		readingAnchorQueue[memberId] = max(eventId, val)
	} else {
		readingAnchorQueue[memberId] = eventId
	}
}

func FlushReadingAnchor() {
	if len(readingAnchorQueue) == 0 {
		return
	}
	for k, v := range readingAnchorQueue {
		if err := database.C.Model(&models.ChannelMember{}).
			Where("id = ?", k).
			Updates(map[string]any{
				"reading_anchor": gorm.Expr("GREATEST(COALESCE(reading_anchor, 0), ?)", v),
			}).Error; err != nil {
			log.Error().Err(err).Msg("An error occurred when flushing reading anchor...")
			return
		}
	}
	clear(readingAnchorQueue)
}
