package server

import (
	"sync"

	"go-ids/internal/db"

	"io"

	"github.com/gin-gonic/gin"
)

// EventManager handles SSE clients
type EventManager struct {
	Message       chan db.Alert
	NewClients    chan chan db.Alert
	ClosedClients chan chan db.Alert
	TotalClients  map[chan db.Alert]bool
	mutex         sync.Mutex
}

var Manager = &EventManager{
	Message:       make(chan db.Alert),
	NewClients:    make(chan chan db.Alert),
	ClosedClients: make(chan chan db.Alert),
	TotalClients:  make(map[chan db.Alert]bool),
}

// Listen starts the event manager listener loop
func (stream *EventManager) Listen() {
	for {
		select {
		case client := <-stream.NewClients:
			stream.mutex.Lock()
			stream.TotalClients[client] = true
			stream.mutex.Unlock()

		case client := <-stream.ClosedClients:
			stream.mutex.Lock()
			delete(stream.TotalClients, client)
			close(client)
			stream.mutex.Unlock()

		case eventMsg := <-stream.Message:
			stream.mutex.Lock()
			for clientMessageChan := range stream.TotalClients {
				select {
				case clientMessageChan <- eventMsg:
				default:
					// Drop message if client is blocked, maybe cleaner to close?
					// For simplicity in SSE, we just skip blocking channels
				}
			}
			stream.mutex.Unlock()
		}
	}
}

// SSEHeaders sets required headers for SSE
func SSEHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
}

// ServerSentEventsHandler handles the SSE endpoint
func ServerSentEventsHandler(c *gin.Context) {
	SSEHeaders(c)

	clientChan := make(chan db.Alert)
	Manager.NewClients <- clientChan

	defer func() {
		Manager.ClosedClients <- clientChan
	}()

	// Send an initial ping or status
	// c.SSEvent("message", "connected") // Optional

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-clientChan:
			c.SSEvent("alert", msg)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
