package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"helpdesk/backend/internal/domain"
	"helpdesk/backend/internal/repository"
	"helpdesk/backend/internal/service"
)

type API struct {
	service *service.Service
	secret  []byte
}

func New(s *service.Service, secret string) *API { return &API{service: s, secret: []byte(secret)} }

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", a.register)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.Handle("GET /api/me", a.auth(http.HandlerFunc(a.me)))
	mux.Handle("GET /api/tickets", a.auth(http.HandlerFunc(a.listTickets)))
	mux.Handle("POST /api/tickets", a.auth(http.HandlerFunc(a.createTicket)))
	mux.Handle("GET /api/tickets/{id}", a.auth(http.HandlerFunc(a.ticketDetail)))
	mux.Handle("PATCH /api/tickets/{id}", a.auth(http.HandlerFunc(a.updateTicket)))
	mux.Handle("POST /api/tickets/{id}/comments", a.auth(http.HandlerFunc(a.addComment)))
	mux.Handle("GET /api/users", a.auth(http.HandlerFunc(a.listUsers)))
	mux.Handle("GET /api/agents", a.auth(http.HandlerFunc(a.listAgents)))
	mux.Handle("PATCH /api/users/{id}/role", a.auth(http.HandlerFunc(a.changeRole)))
	return recoverer(mux)
}

type contextKey struct{}

func userFrom(r *http.Request) domain.User {
	u, _ := r.Context().Value(contextKey{}).(domain.User)
	return u
}

func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.Fields(r.Header.Get("Authorization"))
		if len(header) != 2 || !strings.EqualFold(header[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		id, err := service.ParseToken(header[1], a.secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, err := a.service.CurrentUser(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, user)))
	})
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Email, Password string }
	if !decode(w, r, &req) {
		return
	}
	token, user, err := a.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": user})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if !decode(w, r, &req) {
		return
	}
	token, user, err := a.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFrom(r))
}

func (a *API) createTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
		AssigneeID  *int64 `json:"assigneeId"`
	}
	if !decode(w, r, &req) {
		return
	}
	t, err := a.service.CreateTicket(r.Context(), userFrom(r), service.TicketInput{
		Title: req.Title, Description: req.Description, Category: req.Category, Priority: req.Priority, AssigneeID: req.AssigneeID,
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ticket": t})
}

func (a *API) listTickets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := queryInt(q.Get("page"), 1)
	if err != nil || page < 1 {
		writeError(w, http.StatusBadRequest, "page must be a positive integer")
		return
	}
	pageSize, err := queryInt(q.Get("pageSize"), 20)
	if err != nil || pageSize < 1 || pageSize > 100 {
		writeError(w, http.StatusBadRequest, "pageSize must be between 1 and 100")
		return
	}
	offset64 := int64(page-1) * int64(pageSize)
	if offset64 > 2_000_000_000 {
		writeError(w, http.StatusBadRequest, "page is too large")
		return
	}
	assigneeQuery := q.Get("assignee")
	if assigneeQuery == "" {
		assigneeQuery = q.Get("assigneeId")
	}
	assigneeUnassigned := assigneeQuery == "unassigned"
	assignee := int64(0)
	if assigneeQuery != "" && !assigneeUnassigned {
		assignee, err = strconv.ParseInt(assigneeQuery, 10, 64)
		if err != nil || assignee < 1 {
			writeError(w, http.StatusBadRequest, "assignee must be a positive integer or unassigned")
			return
		}
	}
	if assigneeUnassigned && q.Get("assigneeId") != "" {
		writeError(w, http.StatusBadRequest, "use either assignee or assigneeId, not both")
		return
	}
	sort := q.Get("sort")
	if sort == "" {
		sort = q.Get("sortBy")
	}
	if sort != "" && sort != "createdAt" && sort != "updatedAt" && sort != "title" && sort != "priority" && sort != "status" {
		writeError(w, http.StatusBadRequest, "unsupported sort field")
		return
	}
	direction := q.Get("direction")
	if direction == "" {
		direction = q.Get("sortOrder")
	}
	if direction == "" {
		direction = "desc"
	}
	if direction != "asc" && direction != "desc" {
		writeError(w, http.StatusBadRequest, "direction must be asc or desc")
		return
	}
	title := q.Get("search")
	if title == "" {
		title = q.Get("q")
	}
	if title == "" {
		title = q.Get("title")
	}
	items, total, err := a.service.ListTickets(r.Context(), userFrom(r), repository.TicketFilter{
		Status: q.Get("status"), Priority: q.Get("priority"), AssigneeID: assignee, AssigneeUnassigned: assigneeUnassigned, Title: title,
		Sort: sort, Direction: direction, Limit: pageSize, Offset: int(offset64),
	})
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "pageSize": pageSize, "total": total})
}

func (a *API) ticketDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, comments, err := a.service.TicketDetail(r.Context(), userFrom(r), id)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": t, "comments": comments})
}

func (a *API) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title       *string         `json:"title"`
		Description *string         `json:"description"`
		Category    *string         `json:"category"`
		Priority    *string         `json:"priority"`
		Status      *string         `json:"status"`
		AssigneeID  json.RawMessage `json:"assigneeId"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	allowed := map[string]bool{"title": true, "description": true, "category": true, "priority": true, "status": true, "assigneeId": true}
	for key := range fields {
		if !allowed[key] {
			writeError(w, http.StatusBadRequest, "unknown field: "+key)
			return
		}
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var assignee *int64
	provided := map[string]bool{}
	for key, value := range fields {
		provided[key] = true
		if key != "assigneeId" && string(value) == "null" {
			writeError(w, http.StatusBadRequest, key+" cannot be null")
			return
		}
	}
	if len(req.AssigneeID) > 0 {
		if string(req.AssigneeID) != "null" {
			if err := json.Unmarshal(req.AssigneeID, &assignee); err != nil || assignee == nil {
				writeError(w, http.StatusBadRequest, "assigneeId must be a positive integer or null")
				return
			}
		}
	}
	t, err := a.service.UpdateTicket(r.Context(), userFrom(r), id, service.TicketPatch{
		Title: req.Title, Description: req.Description, Category: req.Category, Priority: req.Priority,
		Status: req.Status, AssigneeID: assignee, AssigneeProvided: provided["assigneeId"],
	}, provided)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": t})
}

func (a *API) addComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Body       string `json:"body"`
		IsInternal bool   `json:"isInternal"`
	}
	if !decode(w, r, &req) {
		return
	}
	comment, err := a.service.AddComment(r.Context(), userFrom(r), id, req.Body, req.IsInternal)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"comment": comment})
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.service.ListUsers(r.Context(), userFrom(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (a *API) listAgents(w http.ResponseWriter, r *http.Request) {
	users, err := a.service.ListAssignableUsers(r.Context(), userFrom(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (a *API) changeRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Role domain.Role `json:"role"`
	}
	if !decode(w, r, &req) {
		return
	}
	user, err := a.service.ChangeRole(r.Context(), userFrom(r), id, req.Role)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (a *API) fail(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalid), errors.Is(err, service.ErrConflict), errors.Is(err, service.ErrInvalidTransition):
		status = http.StatusBadRequest
		if errors.Is(err, service.ErrConflict) {
			status = http.StatusConflict
		}
	case errors.Is(err, service.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	default:
		log.Printf("request failed: %v", err)
	}
	writeError(w, status, service.ErrorMessage(err))
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func queryInt(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
func queryInt64(value string, fallback int64) (int64, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseInt(value, 10, 64)
}
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic serving request: %v", recovered)
				writeError(w, http.StatusInternalServerError, fmt.Sprint("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
