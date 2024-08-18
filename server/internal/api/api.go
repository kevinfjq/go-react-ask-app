package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/kevinfjq/go-react-ask-app/server/internal/store/pgstore"
	"log/slog"
	"net/http"
	"sync"
)

type apiHandler struct {
	q           *pgstore.Queries
	r           *chi.Mux
	upgrader    websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mu          *sync.Mutex
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q:           q,
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mu:          &sync.Mutex{},
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/subscribe/{room_id}", a.handleSubscribe)

	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", a.handleCreateRoom)
			r.Get("/", a.handleGetRooms)

			r.Route("/{room_id}/messages", func(r chi.Router) {
				r.Post("/", a.handleCreateMessage)
				r.Get("/", a.handleGetRoomMessages)

				r.Route("/{message_id}", func(r chi.Router) {
					r.Get("/", a.handleGetRoomMessage)
					r.Patch("/react", a.handleReactToMessage)
					r.Delete("/react", a.handleRemoveReactFromMessage)
					r.Patch("/answer", a.handleMarkMessageAsAnswered)
				})
			})
		})
	})
	a.r = r
	return a
}

const (
	MessageKindMessageCreated           = "message_created"
	MessageKindMessageReactionIncreased = "message_reaction_increased"
	MessageKindMessageReactionDecreased = "message_reaction_decreased"
	MessageKindMessageAnswered          = "message_answered"
)

type MessageMessageReactionIncreased struct {
	Id    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageReactionDecreased struct {
	Id    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageAnswered struct {
	Id string `json:"id"`
}

type MessageMessageCreated struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind   string `json:"kind"`
	Value  any    `json:"value"`
	RoomId string `json:"-"`
}

func (h apiHandler) notifyClients(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subscribers, ok := h.subscribers[msg.RoomId]
	if !ok || len(subscribers) == 0 {
		return
	}

	for conn, cancel := range subscribers {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Error("failed to send message to client", "error", err)
			cancel()
		}
	}
}

func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	type _body struct {
		Theme string `json:"theme"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid json"), http.StatusBadRequest)
		return
	}
	roomId, err := h.q.InsertRoom(r.Context(), body.Theme)
	if err != nil {
		slog.Error("failed to insert room", "error", err)
		http.Error(w, fmt.Sprintf("something went wrong"), http.StatusBadRequest)
		return
	}
	type response struct {
		Id string `json:"id"`
	}
	h.sendJSON(w, response{Id: roomId.String()})
}
func (h apiHandler) handleGetRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.q.GetRooms(r.Context())
	if err != nil {
		slog.Error("failed to get  rooms", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	if rooms == nil {
		rooms = []pgstore.Room{}
	}
	h.sendJSON(w, rooms)
}

func (h apiHandler) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomId, roomId, ok := h.readRoom(w, r)
	if !ok {
		return
	}
	type _body struct {
		Message string `json:"message"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid json"), http.StatusBadRequest)
		return
	}
	messageId, err := h.q.InsertMessage(r.Context(), pgstore.InsertMessageParams{RoomID: roomId, Message: body.Message})
	if err != nil {
		slog.Error("failed to insert message to room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	type response struct {
		Id string `json:"id"`
	}
	h.sendJSON(w, response{Id: messageId.String()})

	go h.notifyClients(Message{
		Kind: MessageKindMessageCreated,
		Value: MessageMessageCreated{
			Id:      messageId.String(),
			Message: body.Message,
		},
		RoomId: rawRoomId,
	})
}

func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request) {
	_, _, roomId, ok := h.readRoom(w, r)
	if !ok {
		return
	}
	messages, err := h.q.GetRoomMessages(r.Context(), roomId)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to get room messages", "error", err)
		return
	}

	if messages == nil {
		messages = []pgstore.Message{}
	}
	h.sendJSON(w, messages)
}
func (h apiHandler) handleGetRoomMessage(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}
	rawMessageId := chi.URLParam(r, "message_id")
	messageId, err := uuid.Parse(rawMessageId)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	message, err := h.q.GetMessage(r.Context(), messageId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "message not found", http.StatusBadRequest)
			return
		}

		slog.Error("failed to get message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	h.sendJSON(w, message)
}
func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomId, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}
	rawMessageId := chi.URLParam(r, "message_id")
	messageId, err := uuid.Parse(rawMessageId)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	count, err := h.q.ReactToMessage(r.Context(), messageId)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to react to message", "error", err)
		return
	}
	type response struct {
		Count int64 `json:"count"`
	}
	h.sendJSON(w, response{Count: count})

	go h.notifyClients(Message{
		Kind:   MessageKindMessageReactionIncreased,
		RoomId: rawRoomId,
		Value: MessageMessageReactionIncreased{
			Count: count,
			Id:    rawMessageId,
		},
	})
}
func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomId, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	rawMessageId := chi.URLParam(r, "message_id")
	messageId, err := uuid.Parse(rawMessageId)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	count, err := h.q.RemoveReactionFromMessage(r.Context(), messageId)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to remove reaction from message", "error", err)
		return
	}
	type response struct {
		Count int64 `json:"count"`
	}
	h.sendJSON(w, response{Count: count})

	go h.notifyClients(Message{
		RoomId: rawRoomId,
		Kind:   MessageKindMessageReactionDecreased,
		Value: MessageMessageReactionDecreased{
			Id:    rawMessageId,
			Count: count,
		},
	})
}
func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request) {
	_, rawRoomId, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}
	rawMessageId := chi.URLParam(r, "message_id")
	messageId, err := uuid.Parse(rawMessageId)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	err = h.q.MarkMessageAsAnswered(r.Context(), messageId)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to mark message as answered", "error", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	go h.notifyClients(Message{
		RoomId: rawRoomId,
		Kind:   MessageKindMessageAnswered,
		Value: MessageMessageAnswered{
			Id: rawMessageId,
		},
	})
}

func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	_, rawRoomId, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("failed to upgrade connection", "error", err)
		http.Error(w, fmt.Sprintf("failed to upgrade to ws connection"), http.StatusBadRequest)
		return
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(r.Context())

	h.mu.Lock()
	if _, ok := h.subscribers[rawRoomId]; !ok {
		h.subscribers[rawRoomId] = make(map[*websocket.Conn]context.CancelFunc)
	}
	slog.Info("new client connected", "room_id", rawRoomId, "client_ip", r.RemoteAddr)
	h.subscribers[rawRoomId][c] = cancel
	h.mu.Unlock()

	<-ctx.Done()

	h.mu.Lock()
	delete(h.subscribers[rawRoomId], c)
	h.mu.Unlock()
}
