package dispatch

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type WSMessageType byte

const (
	_ WSMessageType = iota
	WSText
	WSBinary
)

type WebSocketMessage struct {
	Data []byte
	Type WSMessageType
}

func WebSocketTextMessage(msg string) WebSocketMessage {
	return WebSocketMessage{
		Data: []byte(msg),
		Type: WSText,
	}
}
func WebSocketBinaryMessage(msg []byte) WebSocketMessage {
	return WebSocketMessage{
		Data: msg,
		Type: WSBinary,
	}
}

type ServerSentEventMessage struct {
	Event string
	Data  []byte
}

type ProtocolRouter[R RequestAdapter] struct {
	Http             typedHandler[R]
	WebSocket        func(r R, in <-chan WebSocketMessage) <-chan WebSocketMessage
	ServerSentEvents func(r R, ctx context.Context) <-chan ServerSentEventMessage
}

func (proto *ProtocolRouter[R]) AsTypedHandler(logger Printer) typedHandler[R] {
	return func(w http.ResponseWriter, r R) {
		if r.Request().Header.Get("Accept") == "text/event-stream" {
			if proto.ServerSentEvents != nil {
				// set some headers..
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				stream := proto.ServerSentEvents(r, r.Request().Context())

				for msg := range stream {
					if msg.Event > "" {
						fmt.Fprintf(w, "event: %s\n", msg.Event)
					}
					fmt.Fprintf(w, "data: %s\n\n", msg.Data)
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			} else {
				w.WriteHeader(http.StatusNotAcceptable)
			}
			return
		}
		if r.Request().Header.Get("Upgrade") == "websocket" {
			if proto.WebSocket != nil {
				conn, _, _, err := ws.UpgradeHTTP(r.Request(), w)
				if err != nil {
					logger.Printf("Error upgrading websocket connection: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				defer conn.Close()

				in := make(chan WebSocketMessage)
				out := proto.WebSocket(r, in)
				defer func() {
					// Flush the out channel when done.
					for range out {
					}
				}()

				go func() {
					// Read messages from conn, pass along to in
					defer close(in)
					for {
						payload, opcode, err := wsutil.ReadClientData(conn)
						if err != nil {
							if !errors.As(err, new(wsutil.ClosedError)) {
								logger.Printf("Error reading websocket payload: %v", err)
							}
							return
						}

						msgType := WSText
						if opcode != ws.OpText {
							msgType = WSBinary
						}

						in <- WebSocketMessage{
							Data: payload,
							Type: msgType,
						}
					}
				}()
				for msg := range out {
					msgType := ws.OpText
					if msg.Type == WSBinary {
						msgType = ws.OpBinary
					}
					err := wsutil.WriteServerMessage(conn, msgType, msg.Data)
					if err != nil {
						logger.Printf("Error writing websocket message: %v", err)
					}
				}
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		if proto.Http != nil {
			proto.Http(w, r)
		} else {
			// HTTP isn't supported, what about WS + SSE?
			if proto.WebSocket != nil && proto.ServerSentEvents == nil {
				// WS only is supported
				w.WriteHeader(http.StatusUpgradeRequired)
			} else if proto.WebSocket == nil && proto.ServerSentEvents != nil {
				// SSE only is supported
				w.WriteHeader(http.StatusNotAcceptable)
			} else {
				// if either WS or SSE are supported, just not HTTP:
				w.WriteHeader(http.StatusBadRequest)
			}
		}
	}
}
