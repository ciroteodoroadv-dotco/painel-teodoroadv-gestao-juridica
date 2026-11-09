package sse

import (
"fmt"
"net/http"
"strings"
"sync"
)

type Event struct {
Name string
Data string
}

type Hub struct {
clients   map[chan Event]bool
addClient chan chan Event
remClient chan chan Event
broadcast chan Event
mu        sync.Mutex
}

func NewHub() *Hub {
h := &Hub{
clients:   make(map[chan Event]bool),
addClient: make(chan chan Event),
remClient: make(chan chan Event),
broadcast: make(chan Event),
}
go h.run()
return h
}

func (h *Hub) run() {
for {
select {
case client := <-h.addClient:
h.mu.Lock()
h.clients[client] = true
h.mu.Unlock()
case client := <-h.remClient:
h.mu.Lock()
if _, ok := h.clients[client]; ok {
delete(h.clients, client)
close(client)
}
h.mu.Unlock()
case event := <-h.broadcast:
h.mu.Lock()
for client := range h.clients {
select {
case client <- event:
default:
}
}
h.mu.Unlock()
}
}
}

// Broadcast envia uma mensagem SSE para todos os navegadores conectados
func (h *Hub) Broadcast(name, htmlContent string) {
// Remove quebras de linha para manter a sintaxe do protocolo SSE
singleLineContent := strings.ReplaceAll(htmlContent, "\n", "")
h.broadcast <- Event{Name: name, Data: singleLineContent}
}

// ServeHTTP disponibiliza o endpoint SSE /events
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("Access-Control-Allow-Origin", "*")

flusher, ok := w.(http.Flusher)
if !ok {
http.Error(w, "Streaming HTTP não suportado", http.StatusInternalServerError)
return
}

clientChan := make(chan Event, 10)
h.addClient <- clientChan
defer func() {
h.remClient <- clientChan
}()

ctxDone := r.Context().Done()
for {
select {
case <-ctxDone:
return
case event, ok := <-clientChan:
if !ok {
return
}
fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Name, event.Data)
flusher.Flush()
}
}
}
