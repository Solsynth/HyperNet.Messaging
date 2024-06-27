package services

import (
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

func EncodeMessageBody(body models.EventMessageBody) map[string]any {
	var parsed map[string]any
	raw, _ := jsoniter.Marshal(body)
	_ = jsoniter.Unmarshal(raw, &parsed)
	return parsed
}

func EditMessage(event models.Event, body models.EventMessageBody) (models.Event, error) {
	event.Body = EncodeMessageBody(body)
	event, err := EditEvent(event)
	if err != nil {
		return event, err
	}
	body.RelatedEvent = event.ID
	_, err = NewEvent(models.Event{
		Uuid:      uuid.NewString(),
		Body:      EncodeMessageBody(body),
		Type:      models.EventMessageEdit,
		Channel:   event.Channel,
		Sender:    event.Sender,
		ChannelID: event.ChannelID,
		SenderID:  event.SenderID,
	})
	if err != nil {
		return event, err
	}

	return event, nil
}

func DeleteMessage(event models.Event) (models.Event, error) {
	_, err := DeleteEvent(event)
	if err != nil {
		return event, err
	}
	_, err = NewEvent(models.Event{
		Uuid: uuid.NewString(),
		Body: EncodeMessageBody(models.EventMessageBody{
			RelatedEvent: event.ID,
		}),
		Type:      models.EventMessageDelete,
		Channel:   event.Channel,
		Sender:    event.Sender,
		ChannelID: event.ChannelID,
		SenderID:  event.SenderID,
	})
	if err != nil {
		return event, err
	}

	return event, nil
}
