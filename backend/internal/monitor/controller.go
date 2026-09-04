package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

// Controller keeps monitor HTTP handling dependent on authentication and storage contracts.
type Controller struct {
	auth     Authenticator
	store    Store
	favicons FaviconResolver
}

// Authenticator lets protected handlers share token validation without coupling them to auth storage.
type Authenticator interface {
	UserIDFromRequest(*http.Request) (uuid.UUID, error)
}

const (
	maxMonitorBodySize  = 16 << 10
	maxMonitorURLLength = 2048
	minMonitorInterval  = 1
	maxMonitorInterval  = 7 * 24 * 60 * 60
)

// NewController wires monitor HTTP handling to authentication and storage dependencies.
func NewController(authenticator Authenticator, store Store, faviconResolvers ...FaviconResolver) *Controller {
	var favicons FaviconResolver
	if len(faviconResolvers) > 0 {
		favicons = faviconResolvers[0]
	}
	return &Controller{auth: authenticator, store: store, favicons: favicons}
}

// RegisterRoutes exposes only the monitor operations supported by this API version.
func (controller *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/monitors", controller.list)
	mux.HandleFunc("POST /api/v1/monitors", controller.create)
	mux.HandleFunc("PATCH /api/v1/monitors/{id}", controller.update)
	mux.HandleFunc("DELETE /api/v1/monitors/{id}", controller.delete)
}

type createRequest struct {
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

type updateRequest = createRequest

type response struct {
	ID              string `json:"id"`
	URL             string `json:"url"`
	FaviconURL      string `json:"favicon_url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// create registers a URL monitor for the authenticated user.
//
// @Summary Create a monitor
// @Tags monitors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createRequest true "Monitor configuration"
// @Success 201 {object} response
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/monitors [post]
func (controller *Controller) create(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxMonitorBodySize)
	var body createRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || !validMonitorInput(body) {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return
	}
	value := models.Monitor{ID: uuid.New(), UserID: userID, URL: strings.TrimSpace(body.URL), IntervalSeconds: body.IntervalSeconds}
	if err := controller.store.Create(request.Context(), value); err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	controller.refreshFavicon(request.Context(), userID, &value)
	writeJSON(writer, http.StatusCreated, monitorResponse(value))
}

// list returns all monitors owned by the authenticated user.
//
// @Summary List monitors
// @Tags monitors
// @Produce json
// @Security BearerAuth
// @Success 200 {array} response
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/monitors [get]
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
	for index := range monitors {
		value := &monitors[index]
		if value.FaviconPath == "" {
			controller.refreshFavicon(request.Context(), userID, value)
		}
		result = append(result, monitorResponse(*value))
	}
	writeJSON(writer, http.StatusOK, result)
}

// update changes a monitor owned by the authenticated user.
//
// @Summary Update a monitor
// @Tags monitors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Monitor ID"
// @Param body body updateRequest true "Monitor configuration"
// @Success 200 {object} response
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/monitors/{id} [patch]
func (controller *Controller) update(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	monitorID, err := uuid.Parse(request.PathValue("id"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid monitor id")
		return
	}
	body, ok := decodeUpdateRequest(writer, request)
	if !ok {
		return
	}
	value := models.Monitor{ID: monitorID, UserID: userID, URL: strings.TrimSpace(body.URL), IntervalSeconds: body.IntervalSeconds}
	if err := controller.store.Update(request.Context(), userID, value); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(writer, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	controller.refreshFavicon(request.Context(), userID, &value)
	writeJSON(writer, http.StatusOK, monitorResponse(value))
}

// delete removes a monitor owned by the authenticated user.
//
// @Summary Delete a monitor
// @Tags monitors
// @Security BearerAuth
// @Param id path string true "Monitor ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/monitors/{id} [delete]
func (controller *Controller) delete(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	monitorID, err := uuid.Parse(request.PathValue("id"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid monitor id")
		return
	}
	if err := controller.store.Delete(request.Context(), userID, monitorID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(writer, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func decodeUpdateRequest(writer http.ResponseWriter, request *http.Request) (updateRequest, bool) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxMonitorBodySize)
	var body updateRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || !validMonitorInput(body) {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return updateRequest{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return updateRequest{}, false
	}
	return body, true
}

func (controller *Controller) userID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	userID, err := controller.auth.UserIDFromRequest(request)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	return userID, true
}

func validURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxMonitorURLLength {
		return false
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != "" && parsed.User == nil
}

func validMonitorInput(value createRequest) bool {
	return validURL(value.URL) && value.IntervalSeconds >= minMonitorInterval && value.IntervalSeconds <= maxMonitorInterval
}

func monitorResponse(value models.Monitor) response {
	return response{ID: value.ID.String(), URL: value.URL, FaviconURL: value.FaviconPath, IntervalSeconds: value.IntervalSeconds}
}

func (controller *Controller) refreshFavicon(ctx context.Context, userID uuid.UUID, value *models.Monitor) {
	if controller.favicons == nil {
		return
	}
	faviconPath, err := controller.favicons.Fetch(ctx, value.URL, value.ID)
	if err != nil {
		faviconPath = ""
	}
	if err := controller.store.UpdateFavicon(ctx, userID, value.ID, faviconPath); err != nil {
		return
	}
	value.FaviconPath = faviconPath
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
