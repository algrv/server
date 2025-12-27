package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test_websocket.go <session_id> <token>")
		fmt.Println("Example: go run test_websocket.go abc123 jwt_token_here")
		os.Exit(1)
	}

	sessionID := os.Args[1]
	token := os.Args[2]

	// build WebSocket URL
	u := url.URL{
		Scheme: "ws",
		Host:   "localhost:8080",
		Path:   "/api/v1/ws",
	}
	q := u.Query()
	q.Set("session_id", sessionID)
	q.Set("token", token)
	u.RawQuery = q.Encode()

	fmt.Printf("Connecting to %s\n", u.String())

	// connect
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	fmt.Println("✅ Connected to WebSocket!")

	// handle interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	// read messages
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			fmt.Printf("📨 Received: %s\n", message)
		}
	}()

	// send a test code update after connection
	time.Sleep(1 * time.Second)
	codeUpdate := map[string]interface{}{
		"type": "code_update",
		"payload": map[string]interface{}{
			"code": "console.log('Hello from WebSocket!');",
		},
	}
	codeUpdateJSON, _ := json.Marshal(codeUpdate)
	fmt.Printf("📤 Sending code update: %s\n", codeUpdateJSON)
	err = c.WriteMessage(websocket.TextMessage, codeUpdateJSON)
	if err != nil {
		log.Println("write:", err)
		return
	}

	// wait for interrupt or done
	select {
	case <-done:
		return
	case <-interrupt:
		fmt.Println("\n🛑 Interrupt received, closing connection...")

		// cleanly close the connection
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close:", err)
			return
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
