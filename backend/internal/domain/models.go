package domain

import "time"

type Role string

const (
	Requester Role = "requester"
	Agent     Role = "agent"
	Admin     Role = "admin"
)

func ValidRole(role Role) bool { return role == Requester || role == Agent || role == Admin }
func IsStaff(role Role) bool   { return role == Agent || role == Admin }

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type Ticket struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	RequesterID int64     `json:"requesterId"`
	AssigneeID  *int64    `json:"assigneeId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Comment struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticketId"`
	AuthorID   int64     `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	IsInternal bool      `json:"isInternal"`
	CreatedAt  time.Time `json:"createdAt"`
}

func ValidPriority(priority string) bool {
	return priority == "low" || priority == "medium" || priority == "high" || priority == "urgent"
}

func ValidStatus(status string) bool {
	return status == "open" || status == "in_progress" || status == "resolved" || status == "closed"
}

func CanTransition(role Role, from, to string) bool {
	switch {
	case from == "open" && to == "in_progress":
		return IsStaff(role)
	case from == "in_progress" && to == "resolved":
		return IsStaff(role)
	case from == "resolved" && to == "closed":
		return role == Requester || IsStaff(role)
	case from == "resolved" && to == "open":
		return role == Requester
	case from == "open" && to == "closed":
		return role == Requester || role == Admin
	default:
		return false
	}
}

func CanAccessTicket(user User, ticket Ticket) bool {
	return IsStaff(user.Role) || user.Role == Requester && user.ID == ticket.RequesterID
}

func CanDeleteTicket(user User, ticket Ticket) bool {
	return user.Role == Admin || user.Role == Requester && user.ID == ticket.RequesterID && ticket.Status == "open"
}

func CanViewInternalComments(user User) bool { return IsStaff(user.Role) }
