package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/E2klime/HAXinceL2/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ConnectedClient struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan *protocol.Message
	Hostname string
	Username string
	OS       string
	LastSeen time.Time
}

type Server struct {
	clients    map[string]*ConnectedClient
	mutex      sync.RWMutex
	register   chan *ConnectedClient
	unregister chan *ConnectedClient
	broadcast  chan *protocol.Message
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[string]*ConnectedClient),
		register:   make(chan *ConnectedClient),
		unregister: make(chan *ConnectedClient),
		broadcast:  make(chan *protocol.Message),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client.ID] = client
			s.mutex.Unlock()
			log.Printf("Client registered: %s (%s@%s)", client.ID, client.Username, client.Hostname)

		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.ID)
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &ConnectedClient{
		Conn:     conn,
		Send:     make(chan *protocol.Message, 256),
		LastSeen: time.Now(),
	}

	var msg protocol.Message
	if err := conn.ReadJSON(&msg); err != nil {
		log.Printf("Failed to read auth message: %v", err)
		conn.Close()
		return
	}

	if msg.Type != protocol.TypeAuth {
		log.Printf("Expected auth message, got: %s", msg.Type)
		conn.Close()
		return
	}

	var authPayload protocol.AuthPayload
	if err := json.Unmarshal(msg.Payload, &authPayload); err != nil {
		log.Printf("Failed to parse auth payload: %v", err)
		conn.Close()
		return
	}

	client.ID = authPayload.ClientID
	client.Hostname = authPayload.Hostname
	client.Username = authPayload.Username
	client.OS = authPayload.OS

	s.register <- client

	go s.writePump(client)
	go s.readPump(client)
}

func (s *Server) readPump(client *ConnectedClient) {
	defer func() {
		s.unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg protocol.Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		client.LastSeen = time.Now()
	}
}

func (s *Server) writePump(client *ConnectedClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) GetClients() []*ConnectedClient {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	clients := make([]*ConnectedClient, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}

func (s *Server) GetClient(id string) (*ConnectedClient, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	client, ok := s.clients[id]
	if !ok {
		return nil, fmt.Errorf("client not found: %s", id)
	}
	return client, nil
}

func (s *Server) SendCommand(clientID string, msg *protocol.Message) error {
	client, err := s.GetClient(clientID)
	if err != nil {
		return err
	}

	select {
	case client.Send <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending command to client %s", clientID)
	}
}
