package monitor

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"uptime-backend/internal/auth"
	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

// Controller exposes monitoring point operations over HTTP.
type Controller struct {
	service *auth.Service
	store   Store
}

// NewController creates a monitoring controller.
func NewController(service *auth.Service, store Store) *Controller {
	return &Controller{service: service, store: store}
}

// RegisterRoutes adds monitoring routes to a mux.
func (controller *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/monitors", controller.list)
	mux.HandleFunc("POST /api/v1/monitors", controller.create)
}

type createRequest struct {
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

type response struct {
	ID              string `json:"id"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

func (controller *Controller) create(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	var body createRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || !validURL(body.URL) || body.IntervalSeconds <= 0 {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return
	}
	value := models.Monitor{ID: uuid.New(), UserID: userID, URL: strings.TrimSpace(body.URL), IntervalSeconds: body.IntervalSeconds}
	if err := controller.store.Create(request.Context(), value); err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(writer, http.StatusCreated, monitorResponse(value))
}

func (controller *Controller) list(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	monitors, err := controller.store.List(request.Context(), userID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	result := make([]response, 0, len(monitors))
	for _, value := range monitors {
		result = append(result, monitorResponse(value))
	}
	writeJSON(writer, http.StatusOK, result)
}

func (controller *Controller) userID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	userID, err := controller.service.UserIDFromRequest(request)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	return userID, true
}

func validURL(value string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func monitorResponse(value models.Monitor) response {
	return response{ID: value.ID.String(), URL: value.URL, IntervalSeconds: value.IntervalSeconds}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
