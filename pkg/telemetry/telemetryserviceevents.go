package telemetry

import (
	"context"
	"time"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/utils"
	"github.com/livekit/protocol/webhook"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/livekit/livekit-server/pkg/telemetry/prometheus"
)

func (t *telemetryService) RoomStarted(ctx context.Context, room *livekit.Room) {
	prometheus.RoomStarted()

	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event: webhook.EventRoomStarted,
		Room:  room,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:      livekit.AnalyticsEventType_ROOM_CREATED,
		Timestamp: &timestamppb.Timestamp{Seconds: room.CreationTime},
		Room:      room,
	})
}

func (t *telemetryService) RoomEnded(ctx context.Context, room *livekit.Room) {
	prometheus.RoomEnded(time.Unix(room.CreationTime, 0))

	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event: webhook.EventRoomFinished,
		Room:  room,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:      livekit.AnalyticsEventType_ROOM_ENDED,
		Timestamp: timestamppb.Now(),
		RoomSid:   room.Sid,
		Room:      room,
	})
}

func (t *telemetryService) ParticipantJoined(ctx context.Context, room *livekit.Room,
	participant *livekit.ParticipantInfo, clientInfo *livekit.ClientInfo) {
	t.Lock()
	t.workers[participant.Sid] = newStatsWorker(ctx, t, room.Sid, room.Name, participant.Sid)
	t.Unlock()

	prometheus.AddParticipant()

	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event:       webhook.EventParticipantJoined,
		Room:        room,
		Participant: participant,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:        livekit.AnalyticsEventType_PARTICIPANT_JOINED,
		Timestamp:   timestamppb.Now(),
		RoomSid:     room.Sid,
		Participant: participant,
		Room:        room,
		SdkType:     clientInfo.GetSdk(),
	})
}

func (t *telemetryService) ParticipantLeft(ctx context.Context, room *livekit.Room, participant *livekit.ParticipantInfo) {
	t.Lock()
	if w := t.workers[participant.Sid]; w != nil {
		w.Close()
		delete(t.workers, participant.Sid)
	}
	t.Unlock()

	prometheus.SubParticipant()

	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event:       webhook.EventParticipantLeft,
		Room:        room,
		Participant: participant,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:          livekit.AnalyticsEventType_PARTICIPANT_LEFT,
		Timestamp:     timestamppb.Now(),
		RoomSid:       room.Sid,
		ParticipantId: participant.Sid,
		Room:          room,
	})
}

func (t *telemetryService) TrackPublished(ctx context.Context, participantID string, track *livekit.TrackInfo) {
	prometheus.AddPublishedTrack(track.Type.String())

	roomID, roomName := t.getRoomDetails(participantID)
	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:          livekit.AnalyticsEventType_TRACK_PUBLISHED,
		Timestamp:     timestamppb.Now(),
		RoomSid:       roomID,
		ParticipantId: participantID,
		Track:         track,
		Room:          &livekit.Room{Name: roomName},
	})
}

func (t *telemetryService) TrackUnpublished(ctx context.Context, participantID string, track *livekit.TrackInfo, ssrc uint32) {
	roomID := ""
	roomName := ""
	t.RLock()
	w := t.workers[participantID]
	t.RUnlock()
	if w != nil {
		roomID = w.roomID
		w.RemoveBuffer(ssrc)
		roomName = w.roomName
	}

	prometheus.SubPublishedTrack(track.Type.String())

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:          livekit.AnalyticsEventType_TRACK_UNPUBLISHED,
		Timestamp:     timestamppb.Now(),
		RoomSid:       roomID,
		ParticipantId: participantID,
		TrackId:       track.Sid,
		Room:          &livekit.Room{Name: roomName},
	})
}

func (t *telemetryService) TrackSubscribed(ctx context.Context, participantID string, track *livekit.TrackInfo) {
	prometheus.AddSubscribedTrack(track.Type.String())

	roomID, roomName := t.getRoomDetails(participantID)
	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:          livekit.AnalyticsEventType_TRACK_SUBSCRIBED,
		Timestamp:     timestamppb.Now(),
		RoomSid:       roomID,
		ParticipantId: participantID,
		TrackId:       track.Sid,
		Room:          &livekit.Room{Name: roomName},
	})
}

func (t *telemetryService) TrackUnsubscribed(ctx context.Context, participantID string, track *livekit.TrackInfo) {
	prometheus.SubSubscribedTrack(track.Type.String())

	roomID, roomName := t.getRoomDetails(participantID)
	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:          livekit.AnalyticsEventType_TRACK_UNSUBSCRIBED,
		Timestamp:     timestamppb.Now(),
		RoomSid:       roomID,
		ParticipantId: participantID,
		TrackId:       track.Sid,
		Room:          &livekit.Room{Name: roomName},
	})
}

func (t *telemetryService) RecordingStarted(ctx context.Context, ri *livekit.RecordingInfo) {
	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event:         webhook.EventRecordingStarted,
		RecordingInfo: ri,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:        livekit.AnalyticsEventType_RECORDING_STARTED,
		Timestamp:   timestamppb.Now(),
		RecordingId: ri.Id,
		Room:        &livekit.Room{Name: ri.RoomName},
	})
}

func (t *telemetryService) RecordingEnded(ctx context.Context, ri *livekit.RecordingInfo) {
	t.notifyEvent(ctx, &livekit.WebhookEvent{
		Event:         webhook.EventRecordingFinished,
		RecordingInfo: ri,
	})

	t.analytics.SendEvent(ctx, &livekit.AnalyticsEvent{
		Type:        livekit.AnalyticsEventType_RECORDING_ENDED,
		Timestamp:   timestamppb.Now(),
		RecordingId: ri.Id,
		Room:        &livekit.Room{Name: ri.RoomName},
	})
}

func (t *telemetryService) getRoomDetails(participantID string) (string, string) {
	t.RLock()
	w := t.workers[participantID]
	t.RUnlock()
	if w != nil {
		return w.roomID, w.roomName
	}
	return "", ""
}

func (t *telemetryService) notifyEvent(ctx context.Context, event *livekit.WebhookEvent) {
	if t.notifier == nil {
		return
	}

	event.CreatedAt = time.Now().Unix()
	event.Id = utils.NewGuid("EV_")

	t.webhookPool.Submit(func() {
		if err := t.notifier.Notify(ctx, event); err != nil {
			logger.Warnw("failed to notify webhook", err, "event", event.Event)
		}
	})
}
