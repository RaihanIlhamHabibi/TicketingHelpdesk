package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"helpdesk/backend/internal/domain"
	"helpdesk/backend/internal/repository"
)

var (
	ErrInvalid           = errors.New("invalid input")
	ErrUnauthorized      = errors.New("invalid credentials")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Store interface {
	CreateUser(context.Context, string, string, string, domain.Role) (domain.User, error)
	UserByEmail(context.Context, string) (domain.User, string, error)
	UserByID(context.Context, int64) (domain.User, error)
	Users(context.Context) ([]domain.User, error)
	UpdateRole(context.Context, int64, domain.Role) (domain.User, error)
	CreateTicket(context.Context, domain.Ticket) (domain.Ticket, error)
	ListTickets(context.Context, repository.TicketFilter) ([]domain.Ticket, int, error)
	TicketByID(context.Context, int64) (domain.Ticket, error)
	UpdateTicket(context.Context, domain.Ticket) (domain.Ticket, error)
	Comments(context.Context, int64, bool) ([]domain.Comment, error)
	CreateComment(context.Context, int64, int64, string, bool) (domain.Comment, error)
}

type Service struct {
	store     Store
	jwtSecret []byte
	tokenTTL  time.Duration
}

func New(store Store, secret string, tokenTTL time.Duration) *Service {
	return &Service{store: store, jwtSecret: []byte(secret), tokenTTL: tokenTTL}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (string, domain.User, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if !utf8.ValidString(name) || len(name) < 2 || len(name) > 100 ||
		!validEmail(email) || len(password) < 8 || len(password) > 72 {
		return "", domain.User{}, ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", domain.User{}, err
	}
	user, err := s.store.CreateUser(ctx, name, email, string(hash), domain.Requester)
	if repository.IsConflict(err) {
		return "", domain.User{}, ErrConflict
	}
	if err != nil {
		return "", domain.User{}, err
	}
	token, err := s.issueToken(user)
	return token, user, err
}

func (s *Service) Login(ctx context.Context, email, password string) (string, domain.User, error) {
	user, hash, err := s.store.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", domain.User{}, ErrUnauthorized
	}
	token, err := s.issueToken(user)
	return token, user, err
}

func (s *Service) CurrentUser(ctx context.Context, id int64) (domain.User, error) {
	user, err := s.store.UserByID(ctx, id)
	if repository.IsNotFound(err) {
		return domain.User{}, ErrUnauthorized
	}
	return user, err
}

func (s *Service) ListUsers(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if actor.Role != domain.Admin {
		return nil, ErrForbidden
	}
	return s.store.Users(ctx)
}

func (s *Service) ListAssignableUsers(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if !domain.IsStaff(actor.Role) {
		return nil, ErrForbidden
	}
	users, err := s.store.Users(ctx)
	if err != nil {
		return nil, err
	}
	assignable := make([]domain.User, 0, len(users))
	for _, user := range users {
		if domain.IsStaff(user.Role) {
			assignable = append(assignable, user)
		}
	}
	return assignable, nil
}

func (s *Service) ChangeRole(ctx context.Context, actor domain.User, id int64, role domain.Role) (domain.User, error) {
	if actor.Role != domain.Admin {
		return domain.User{}, ErrForbidden
	}
	if !domain.ValidRole(role) || id < 1 {
		return domain.User{}, ErrInvalid
	}
	user, err := s.store.UpdateRole(ctx, id, role)
	if repository.IsNotFound(err) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

type TicketInput struct {
	Title, Description, Category, Priority string
	AssigneeID                             *int64
}

type TicketPatch struct {
	Title, Description, Category, Priority *string
	Status                                 *string
	AssigneeID                             *int64
	AssigneeProvided                       bool
}

func (s *Service) CreateTicket(ctx context.Context, actor domain.User, in TicketInput) (domain.Ticket, error) {
	if actor.Role != domain.Requester && !domain.IsStaff(actor.Role) {
		return domain.Ticket{}, ErrForbidden
	}
	in.Title, in.Description, in.Category = strings.TrimSpace(in.Title), strings.TrimSpace(in.Description), strings.TrimSpace(in.Category)
	if err := validateTicketFields(in.Title, in.Description, in.Category, in.Priority); err != nil {
		return domain.Ticket{}, err
	}
	if in.AssigneeID != nil && *in.AssigneeID < 1 {
		return domain.Ticket{}, ErrInvalid
	}
	if actor.Role == domain.Requester && in.AssigneeID != nil {
		return domain.Ticket{}, ErrForbidden
	}
	if in.AssigneeID != nil {
		assignee, err := s.store.UserByID(ctx, *in.AssigneeID)
		if repository.IsNotFound(err) {
			return domain.Ticket{}, ErrInvalid
		}
		if err != nil {
			return domain.Ticket{}, err
		}
		if !domain.IsStaff(assignee.Role) {
			return domain.Ticket{}, ErrInvalid
		}
	}
	t := domain.Ticket{Title: in.Title, Description: in.Description, Category: in.Category,
		Priority: in.Priority, RequesterID: actor.ID, AssigneeID: in.AssigneeID}
	return s.store.CreateTicket(ctx, t)
}

func (s *Service) ListTickets(ctx context.Context, actor domain.User, f repository.TicketFilter) ([]domain.Ticket, int, error) {
	if !domain.ValidRole(actor.Role) {
		return nil, 0, ErrForbidden
	}
	if actor.Role == domain.Requester {
		f.RequesterOnly, f.UserID = true, actor.ID
	}
	if f.Status != "" && !domain.ValidStatus(f.Status) || f.Priority != "" && !domain.ValidPriority(f.Priority) {
		return nil, 0, ErrInvalid
	}
	if f.AssigneeID < 0 {
		return nil, 0, ErrInvalid
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 {
		return nil, 0, ErrInvalid
	}
	return s.store.ListTickets(ctx, f)
}

func (s *Service) TicketDetail(ctx context.Context, actor domain.User, id int64) (domain.Ticket, []domain.Comment, error) {
	t, err := s.store.TicketByID(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Ticket{}, nil, ErrNotFound
	}
	if err != nil {
		return domain.Ticket{}, nil, err
	}
	if !domain.CanAccessTicket(actor, t) {
		return domain.Ticket{}, nil, ErrForbidden
	}
	comments, err := s.store.Comments(ctx, id, domain.CanViewInternalComments(actor))
	return t, comments, err
}

func (s *Service) UpdateTicket(ctx context.Context, actor domain.User, id int64, patch TicketPatch, provided map[string]bool) (domain.Ticket, error) {
	t, err := s.store.TicketByID(ctx, id)
	if repository.IsNotFound(err) {
		return domain.Ticket{}, ErrNotFound
	}
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanAccessTicket(actor, t) {
		return domain.Ticket{}, ErrForbidden
	}
	if actor.Role == domain.Requester {
		for key := range provided {
			if key != "status" {
				return domain.Ticket{}, ErrForbidden
			}
		}
	} else if !domain.IsStaff(actor.Role) {
		return domain.Ticket{}, ErrForbidden
	}
	if patch.Status != nil {
		if !domain.ValidStatus(*patch.Status) {
			return domain.Ticket{}, ErrInvalid
		}
		if !domain.CanTransition(actor.Role, t.Status, *patch.Status) {
			return domain.Ticket{}, fmt.Errorf("%w: %s to %s is not allowed for %s", ErrInvalidTransition, t.Status, *patch.Status, actor.Role)
		}
		t.Status = *patch.Status
	}
	if patch.Title != nil {
		t.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		t.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Category != nil {
		t.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.Priority != nil {
		t.Priority = *patch.Priority
	}
	if patch.AssigneeProvided {
		t.AssigneeID = patch.AssigneeID
	}
	if err := validateTicketFields(t.Title, t.Description, t.Category, t.Priority); err != nil {
		return domain.Ticket{}, err
	}
	if t.AssigneeID != nil && *t.AssigneeID < 1 {
		return domain.Ticket{}, ErrInvalid
	}
	if patch.AssigneeProvided && t.AssigneeID != nil {
		assignee, err := s.store.UserByID(ctx, *t.AssigneeID)
		if repository.IsNotFound(err) {
			return domain.Ticket{}, ErrInvalid
		}
		if err != nil {
			return domain.Ticket{}, err
		}
		if !domain.IsStaff(assignee.Role) {
			return domain.Ticket{}, ErrInvalid
		}
	}
	return s.store.UpdateTicket(ctx, t)
}

func (s *Service) AddComment(ctx context.Context, actor domain.User, ticketID int64, body string, internal bool) (domain.Comment, error) {
	t, err := s.store.TicketByID(ctx, ticketID)
	if repository.IsNotFound(err) {
		return domain.Comment{}, ErrNotFound
	}
	if err != nil {
		return domain.Comment{}, err
	}
	if !domain.CanAccessTicket(actor, t) {
		return domain.Comment{}, ErrForbidden
	}
	if internal && !domain.IsStaff(actor.Role) {
		return domain.Comment{}, ErrForbidden
	}
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 10000 {
		return domain.Comment{}, ErrInvalid
	}
	return s.store.CreateComment(ctx, ticketID, actor.ID, body, internal)
}

func (s *Service) issueToken(user domain.User) (string, error) {
	now := time.Now()
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(map[string]any{"sub": user.ID, "iat": now.Unix(), "exp": now.Add(s.tokenTTL).Unix()})
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.jwtSecret)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func ParseToken(token string, secret []byte) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, ErrUnauthorized
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrUnauthorized
	}
	var claims struct {
		Subject json.RawMessage `json:"sub"`
		Expires int64           `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Expires <= time.Now().Unix() {
		return 0, ErrUnauthorized
	}
	var id int64
	if err := json.Unmarshal(claims.Subject, &id); err != nil || id < 1 {
		text := strings.Trim(string(claims.Subject), `"`)
		id, err = strconv.ParseInt(text, 10, 64)
		if err != nil || id < 1 {
			return 0, ErrUnauthorized
		}
	}
	return id, nil
}

func validateTicketFields(title, description, category, priority string) error {
	if title == "" || len(title) > 200 || description == "" || len(description) > 20000 ||
		category == "" || len(category) > 100 || !domain.ValidPriority(priority) {
		return ErrInvalid
	}
	return nil
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	at := strings.LastIndexByte(email, '@')
	return at > 0 && at < len(email)-3 && strings.Contains(email[at+1:], ".")
}

func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalid):
		return "invalid input"
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, ErrNotFound):
		return "not found"
	case errors.Is(err, ErrConflict):
		return "resource already exists"
	case errors.Is(err, ErrInvalidTransition):
		return err.Error()
	default:
		return fmt.Sprintf("request failed: %v", err)
	}
}
