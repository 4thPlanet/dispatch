package routes

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/4thPlanet/dispatch"
)

func chatHandler(handler *dispatch.TypedHandler[*Handler]) {
	listeners := map[chan<- dispatch.WebSocketMessage]struct{}{}
	mu := sync.RWMutex{}
	chat := dispatch.ProtocolRouter[*Handler]{
		Http: func(w http.ResponseWriter, r *Handler) {
			// output a basic HTML page with chat..
			if _, err := w.Write([]byte(`<html>
      <head>
        <title>Chat Room</title>
        <script type="text/javascript">
          const socket = new WebSocket("ws://"+location.host+"/chat")
          socket.addEventListener("open", () => {
            document.getElementById("message").removeAttribute("disabled")
          })
          socket.addEventListener("message", evt => {
            let m = document.createElement("p")
            m.innerText = evt.data
            document.getElementById("messages").append(m)
          })
          socket.addEventListener("close", () => {
            document.getElementById("message").disabled = "disabled"
          })

          function send() {
            socket.send(document.getElementById("message").value)
            document.getElementById("message").value = ""
          }
        </script>
      </head>
      <body>
        <div id="messages"></div>
        <form onsubmit="send(); return false;">
          <input id="message" name="message" disabled="disabled" />
          <button type="submit">Send</button>
        </form>
      </body>
      </html>`)); err != nil {
				log.Println("Error writing to client: ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
		WebSocket: func(r *Handler, in <-chan dispatch.WebSocketMessage) <-chan dispatch.WebSocketMessage {
			out := make(chan dispatch.WebSocketMessage)
			mu.Lock()
			listeners[out] = struct{}{}
			mu.Unlock()
			go func() {
				defer func() {
					mu.Lock()
					delete(listeners, out)
					close(out)
					mu.Unlock()
				}()

				for msg := range in {
					if msg.Type == dispatch.WSText {
						// send this message to everyone in the chatroom
						mu.RLock()
						for user, _ := range listeners {
							user <- msg
						}
						mu.RUnlock()
					}
				}
			}()
			return out
		},
	}
	wallflower := dispatch.ProtocolRouter[*Handler]{
		Http: func(w http.ResponseWriter, r *Handler) {
			// output a basic HTML page with chat..
			if _, err := w.Write([]byte(`<html>
      <head>
        <title>Chat Room (Wallflower)</title>
        <script type="text/javascript">
          const source = new EventSource("")
          source.addEventListener("Chat", evt => {
            let m = document.createElement("p")
            m.innerText = evt.data
            document.getElementById("messages").append(m)
          })
        </script>
      </head>
      <body>
        <div id="messages"></div>
      </body>
      </html>`)); err != nil {
				log.Println("Error writing to client: ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
		ServerSentEvents: func(r *Handler, ctx context.Context) <-chan dispatch.ServerSentEventMessage {
			chats := make(chan dispatch.WebSocketMessage)

			out := make(chan dispatch.ServerSentEventMessage)
			mu.Lock()
			listeners[chats] = struct{}{}
			mu.Unlock()

			go func() {
				defer func() {
					mu.Lock()
					delete(listeners, chats)
					close(chats)
					mu.Unlock()
					close(out)
				}()
				done := ctx.Done()
				for {
					select {
					case <-done:
						return
					case msg := <-chats:
						out <- dispatch.ServerSentEventMessage{
							Event: "Chat",
							Data:  msg.Data,
						}
					}
				}

			}()

			return out

		},
	}

	handler.HandleFunc("/chat", chat.AsTypedHandler(log.Default()))
	handler.HandleFunc("/chat/wall", wallflower.AsTypedHandler(log.Default()))
}
