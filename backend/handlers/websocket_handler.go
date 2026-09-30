package handlers

import (
	"coding-platform/services"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins - CORS is handled by Gin middleware
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

// WebSocketHandler handles WebSocket connection upgrades
func WebSocketHandler(c *gin.Context) {
	// Get user info from context (set by AuthMiddleware)
	regdNoVal, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userIDStr, ok := regdNoVal.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return
	}

	collegeIDVal, exists := c.Get("college_id")
	if !exists || collegeIDVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "College ID not found"})
		return
	}

	collegeIDStr := ""
	switch v := collegeIDVal.(type) {
	case string:
		collegeIDStr = v
	case *string:
		if v != nil {
			collegeIDStr = *v
		}
	}

	if collegeIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "College ID not found"})
		return
	}

	roleVal, _ := c.Get("role")
	userRoleStr := ""
	if role, ok := roleVal.(string); ok {
		userRoleStr = role
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket: Failed to upgrade connection: %v", err)
		return
	}

	// Get the WebSocket hub
	hub := services.GetWebSocketHub()

	// Register the connection
	hub.Register(conn, userIDStr, collegeIDStr, userRoleStr)

	// Start ping loop to keep connection alive
	done := make(chan struct{})
	go hub.StartPingLoop(conn, done)

	// Cleanup on disconnect
	defer func() {
		close(done)
		hub.Unregister(userIDStr, collegeIDStr)
		conn.Close()
	}()

	// Read loop - handle incoming messages and detect disconnect
	for {
		// Set read deadline for timeout detection
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket: Unexpected close error for user %s: %v", userIDStr, err)
			}
			break
		}

		// Handle PONG responses (client responding to our PINGs)
		if messageType == websocket.TextMessage {
			var msg services.WebSocketMessage
			if err := parseJSON(message, &msg); err == nil {
				if msg.Type == "PONG" {
					// Client is alive, continue
					continue
				}
				// Handle other message types here if needed
				log.Printf("WebSocket: Received message type %s from user %s", msg.Type, userIDStr)
			}
		}
	}
}

// parseJSON safely parses JSON bytes into a struct
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// GetWebSocketStats returns WebSocket connection statistics (admin only)
func GetWebSocketStats(c *gin.Context) {
	hub := services.GetWebSocketHub()

	stats := gin.H{
		"total_connections": hub.GetTotalConnectionCount(),
	}

	c.JSON(http.StatusOK, stats)
}
