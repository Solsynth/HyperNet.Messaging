package services

import (
	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
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
	idSet := lo.Uniq(lo.Map(lo.Keys(readingAnchorQueue), func(item uint, _ int) uint {
		return item
	}))
	var pairs []map[string]any
	for k, v := range readingAnchorQueue {
		pairs = append(pairs, map[string]any{
			"id":             k,
			"reading_anchor": gorm.Expr("GREATEST(reading_anchor, ?)", v),
		})
	}
	if err := database.C.Model(&models.ChannelMember{}).
		Where("id IN ?", idSet).
		Updates(pairs).Error; err != nil {
		log.Error().Err(err).Msg("An error occurred when flushing reading anchor...")
	}
}
