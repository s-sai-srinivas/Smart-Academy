package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Data interface{} `json:"data,omitempty"`
	Type string      `json:"type"`
}

// ClientConnection represents a connected WebSocket client
type ClientConnection struct {
	Conn      *websocket.Conn
	UserID    string
	CollegeID string
	UserRole  string
}

// WebSocketHub manages all WebSocket connections and broadcasts
type WebSocketHub struct {
	collegeConnections map[string][]*ClientConnection
	userConnections    map[string]*ClientConnection
	broadcast          chan broadcastMessage
	mu                 sync.RWMutex
}

// broadcastMessage represents a message to be broadcast
type broadcastMessage struct {
	message   WebSocketMessage
	collegeID string
}

var (
	hubInstance *WebSocketHub
	hubOnce     sync.Once
)

// GetWebSocketHub returns the singleton WebSocket hub instance
func GetWebSocketHub() *WebSocketHub {
	hubOnce.Do(func() {
		hubInstance = &WebSocketHub{
			collegeConnections: make(map[string][]*ClientConnection),
			userConnections:    make(map[string]*ClientConnection),
			broadcast:          make(chan broadcastMessage, 100),
		}
		go hubInstance.runBroadcastLoop()
	})
	return hubInstance
}

// runBroadcastLoop processes broadcast messages
func (h *WebSocketHub) runBroadcastLoop() {
	for msg := range h.broadcast {
		h.broadcastToCollege(msg.collegeID, msg.message)
	}
}

// Register adds a new WebSocket connection to the hub
func (h *WebSocketHub) Register(conn *websocket.Conn, userID, collegeID, userRole string) {
	client := &ClientConnection{
		Conn:      conn,
		UserID:    userID,
		CollegeID: collegeID,
		UserRole:  userRole,
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to college connections
	h.collegeConnections[collegeID] = append(h.collegeConnections[collegeID], client)

	// Add to user connections (for direct messaging)
	h.userConnections[userID] = client

	log.Printf("WebSocket: User %s connected (college: %s, role: %s)", userID, collegeID, userRole)
}

// Unregister removes a WebSocket connection from the hub
func (h *WebSocketHub) Unregister(userID, collegeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from user connections
	delete(h.userConnections, userID)

	// Remove from college connections
	connections := h.collegeConnections[collegeID]
	for i, conn := range connections {
		if conn.UserID == userID {
			// Remove this connection
			h.collegeConnections[collegeID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	// Clean up empty slices
	if len(h.collegeConnections[collegeID]) == 0 {
		delete(h.collegeConnections, collegeID)
	}

	log.Printf("WebSocket: User %s disconnected (college: %s)", userID, collegeID)
}

// BroadcastToCollege sends a message to all connections of a specific college
func (h *WebSocketHub) BroadcastToCollege(collegeID string, message WebSocketMessage) {
	// Use the broadcast channel for async processing
	h.broadcast <- broadcastMessage{
		collegeID: collegeID,
		message:   message,
	}
}

// broadcastToCollege is the internal sync broadcast method
func (h *WebSocketHub) broadcastToCollege(collegeID string, message WebSocketMessage) {
	h.mu.RLock()
	connections := h.collegeConnections[collegeID]
	h.mu.RUnlock()

	if len(connections) == 0 {
		log.Printf("WebSocket: No connections for college %s", collegeID)
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal message: %v", err)
		return
	}

	log.Printf("WebSocket: Broadcasting %s to %d connections in college %s", message.Type, len(connections), collegeID)

	for _, conn := range connections {
		err := conn.Conn.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			log.Printf("WebSocket: Failed to send message to user %s: %v", conn.UserID, err)
			// Connection will be cleaned up by the read loop
		}
	}
}

// SendToUser sends a message to a specific user
func (h *WebSocketHub) SendToUser(userID string, message WebSocketMessage) error {
	h.mu.RLock()
	client, exists := h.userConnections[userID]
	h.mu.RUnlock()

	if !exists {
		return nil // User not connected
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return client.Conn.WriteMessage(websocket.TextMessage, messageBytes)
}

// ForceLogoutCollege sends a FORCE_LOGOUT message to all users of a college
func (h *WebSocketHub) ForceLogoutCollege(collegeID string, reason string) {
	message := WebSocketMessage{
		Type: "FORCE_LOGOUT",
		Data: map[string]string{
			"reason":  reason,
			"message": "Your college has been suspended. Please contact your administrator.",
		},
	}
	h.BroadcastToCollege(collegeID, message)
}

// GetConnectionCount returns the number of active connections for a college
func (h *WebSocketHub) GetConnectionCount(collegeID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.collegeConnections[collegeID])
}

// GetTotalConnectionCount returns the total number of active connections
func (h *WebSocketHub) GetTotalConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userConnections)
}

// StartPingLoop starts a ping loop to keep connections alive
func (h *WebSocketHub) StartPingLoop(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"PING"}`))
			if err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
