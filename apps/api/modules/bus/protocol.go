package bus

import (
	"encoding/json"
	"log/slog"
	"time"
)

func (s *Server) readLoop(conn *connection) {
	defer func() {
		close(conn.closed)
		s.detach(conn)
		conn.socket.Close()
		s.log.Info("pool app disconnected",
			slog.String("app", conn.app.App),
			slog.String("app_id", conn.app.AppID))
		s.report(Activity{Type: "pool.disconnected", App: conn.app.App, Detail: conn.app.App + " left the pool"})
	}()

	for {
		_, raw, err := conn.socket.ReadMessage()
		if err != nil {
			return
		}
		s.handleFrame(conn, raw)
		// Refreshed after the frame is handled, not before: a subscribe replays
		// the backlog inline, and every send in it can block for writeTimeout,
		// so a large replay would otherwise burn the whole budget and the next
		// read would fail the instant the client was finally caught up.
		//
		// Any frame counts, not just a pong — the Go client pings in the
		// protocol rather than in control frames.
		conn.socket.SetReadDeadline(time.Now().Add(pongTimeout))
	}
}

func (s *Server) handleFrame(conn *connection, raw []byte) {
	var frame struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		s.sendError(conn, "PARSE_ERROR", "invalid json")
		return
	}

	switch frame.Type {
	case "subscribe":
		s.handleSubscribe(conn, raw)
	case "event":
		s.handleEvent(conn, raw)
	case "ack":
		s.handleAck(conn, raw)
	case "ping":
		conn.send(map[string]string{"type": "pong"})
	default:
		s.sendError(conn, "UNKNOWN_TYPE", "unknown message type: "+frame.Type)
	}
}

func (s *Server) handleSubscribe(conn *connection, raw []byte) {
	var frame struct {
		Channels map[string]struct {
			LastOffset int64 `json:"last_offset"`
		} `json:"channels"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		s.sendError(conn, "PARSE_ERROR", "invalid subscribe frame")
		return
	}

	confirmed := map[string]map[string]int64{}
	for channel, options := range frame.Channels {
		s.subscribe(conn, channel)
		cursor := s.replay(conn, channel, s.resumeFrom(conn, channel, options.LastOffset))
		confirmed[channel] = map[string]int64{"offset": cursor}
	}

	conn.send(map[string]any{"type": "subscribed", "channels": confirmed})
}

// resumeFrom picks whichever cursor is further along: the client's own, or the
// server-side ack cursor that covers a client which lost its offsets.
func (s *Server) resumeFrom(conn *connection, channel string, clientOffset int64) int64 {
	if acked := s.queue.LastAcked(conn.app.AppID, channel); acked > clientOffset {
		return acked
	}
	return clientOffset
}

// replay walks the backlog from cursor in batches and returns how far it got.
// A read that fails stops the replay where it stands rather than resuming past
// a hole, so the client's next subscribe picks up from the same place.
func (s *Server) replay(conn *connection, channel string, cursor int64) int64 {
	for {
		batch, err := s.queue.After(channel, cursor, replayBatch)
		if err != nil {
			s.log.Error("pool replay failed", slog.String("channel", channel), slog.Any("error", err))
			return cursor
		}
		for _, event := range batch {
			// An app never receives its own emissions back.
			if event.Sender == conn.app.App {
				continue
			}
			conn.send(eventFrame(event))
		}
		if len(batch) > 0 {
			cursor = batch[len(batch)-1].Offset
		}
		if len(batch) < replayBatch {
			return cursor
		}
	}
}

func (s *Server) handleEvent(conn *connection, raw []byte) {
	var frame struct {
		Channel string          `json:"channel"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Channel == "" {
		s.sendError(conn, "INVALID_EVENT", "channel is required")
		return
	}
	if len(frame.Payload) == 0 {
		frame.Payload = json.RawMessage("null")
	}

	event, err := s.queue.Push(frame.Channel, frame.Payload, conn.app.App)
	if err != nil {
		s.log.Error("pool push failed", slog.String("channel", frame.Channel), slog.Any("error", err))
		s.sendError(conn, "PUSH_FAILED", "could not queue the event")
		return
	}

	recipients := s.fanOut(conn, event)
	conn.send(map[string]any{"type": "ack", "channel": event.Channel, "offset": event.Offset})

	s.mu.RLock()
	handler := s.eventHandler
	s.mu.RUnlock()
	if handler != nil {
		handler(conn.app.App, frame.Channel, frame.Payload)
	}

	s.report(Activity{
		Type:   "pool.event",
		App:    conn.app.App,
		Detail: conn.app.App + " → " + event.Channel + " (" + pluralRecipients(recipients) + ")",
	})
}

func (s *Server) handleAck(conn *connection, raw []byte) {
	var frame struct {
		Channel string `json:"channel"`
		Offset  int64  `json:"offset"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Channel == "" {
		return
	}
	if err := s.queue.Ack(conn.app.AppID, frame.Channel, frame.Offset); err != nil {
		s.log.Error("pool ack failed", slog.Any("error", err))
	}
	s.queue.Touch(conn.app.AppID)
}
