package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kevinfjq/go-react-ask-app/server/internal/store/pgstore"
	"log/slog"
	"net/http"
)

func (h apiHandler) readRoom(w http.ResponseWriter, r *http.Request) (room pgstore.Room, rawRoomId string, roomId uuid.UUID, ok bool) {
	rawRoomId = chi.URLParam(r, "room_id")
	roomId, err := uuid.Parse(rawRoomId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid room id %s", rawRoomId), http.StatusBadRequest)
		return pgstore.Room{}, "", uuid.UUID{}, false
	}
	room, err = h.q.GetRoom(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, fmt.Sprintf("room not fount"), http.StatusBadRequest)
			return pgstore.Room{}, "", uuid.UUID{}, false
		}
		slog.Error("failed to get room", "error", err)
		http.Error(w, fmt.Sprintf("something went wrong"), http.StatusInternalServerError)
		return pgstore.Room{}, "", uuid.UUID{}, false
	}

	return room, rawRoomId, roomId, true
}

func (h apiHandler) sendJSON(w http.ResponseWriter, rawData any) {
	data, _ := json.Marshal(rawData)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
