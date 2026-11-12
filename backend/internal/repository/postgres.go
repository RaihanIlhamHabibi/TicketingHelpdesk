package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"helpdesk/backend/internal/config"
	"helpdesk/backend/internal/domain"
)

type Postgres struct{ DB *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{DB: pool}, nil
}

func (p *Postgres) Migrate(ctx context.Context) error {
	sql, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	tx, err := p.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, statement := range strings.Split(string(sql), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (p *Postgres) Seed(ctx context.Context, users []config.SeedUser, hash func(string) (string, error)) error {
	for _, u := range users {
		passwordHash, err := hash(u.Password)
		if err != nil {
			return err
		}
		_, err = p.DB.Exec(ctx, `INSERT INTO users(name,email,password_hash,role)
			VALUES($1,$2,$3,$4) ON CONFLICT(email) DO NOTHING`, u.Name, u.Email, passwordHash, u.Role)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) CreateUser(ctx context.Context, name, email, passwordHash string, role domain.Role) (domain.User, error) {
	return scanUser(p.DB.QueryRow(ctx, `INSERT INTO users(name,email,password_hash,role)
		VALUES($1,$2,$3,$4) RETURNING id,name,email,role,created_at`, name, email, passwordHash, role))
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := p.DB.QueryRow(ctx, `SELECT id,name,email,role,created_at,password_hash FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &hash)
	return u, hash, err
}

func (p *Postgres) UserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(p.DB.QueryRow(ctx, `SELECT id,name,email,role,created_at FROM users WHERE id=$1`, id))
}

func (p *Postgres) Users(ctx context.Context) ([]domain.User, error) {
	rows, err := p.DB.Query(ctx, `SELECT id,name,email,role,created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (p *Postgres) UpdateRole(ctx context.Context, id int64, role domain.Role) (domain.User, error) {
	return scanUser(p.DB.QueryRow(ctx, `UPDATE users SET role=$2 WHERE id=$1
		RETURNING id,name,email,role,created_at`, id, role))
}

func (p *Postgres) CreateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	return scanTicket(p.DB.QueryRow(ctx, `INSERT INTO tickets
		(title,description,category,priority,status,requester_id,assignee_id)
		VALUES($1,$2,$3,$4,'open',$5,$6)
		RETURNING id,title,description,category,priority,status,requester_id,assignee_id,created_at,updated_at`,
		ticket.Title, ticket.Description, ticket.Category, ticket.Priority, ticket.RequesterID, ticket.AssigneeID))
}

type TicketFilter struct {
	UserID, AssigneeID                       int64
	RequesterOnly                            bool
	AssigneeUnassigned                       bool
	Status, Priority, Title, Sort, Direction string
	Limit, Offset                            int
}

func (p *Postgres) ListTickets(ctx context.Context, f TicketFilter) ([]domain.Ticket, int, error) {
	conditions := []string{"($1::boolean = false OR requester_id=$2)"}
	args := []any{f.RequesterOnly, f.UserID}
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	if f.Priority != "" {
		add("priority=$%d", f.Priority)
	}
	if f.AssigneeID > 0 {
		add("assignee_id=$%d", f.AssigneeID)
	}
	if f.AssigneeUnassigned {
		conditions = append(conditions, "assignee_id IS NULL")
	}
	if f.Title != "" {
		add("title ILIKE $%d", "%"+f.Title+"%")
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := p.DB.QueryRow(ctx, "SELECT count(*) FROM tickets WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sortColumns := map[string]string{"createdAt": "created_at", "updatedAt": "updated_at", "title": "title", "priority": "priority", "status": "status"}
	sortCol := sortColumns[f.Sort]
	if sortCol == "" {
		sortCol = "created_at"
	}
	direction := "DESC"
	if f.Direction == "asc" {
		direction = "ASC"
	}
	listArgs := append(append([]any{}, args...), f.Limit, f.Offset)
	rows, err := p.DB.Query(ctx, `SELECT id,title,description,category,priority,status,requester_id,assignee_id,created_at,updated_at
		FROM tickets WHERE `+where+fmt.Sprintf(" ORDER BY %s %s,id DESC LIMIT $%d OFFSET $%d", sortCol, direction, len(args)+1, len(args)+2), listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.Ticket{}
	for rows.Next() {
		ticket, err := scanTicket(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, ticket)
	}
	return items, total, rows.Err()
}

func (p *Postgres) TicketByID(ctx context.Context, id int64) (domain.Ticket, error) {
	return scanTicket(p.DB.QueryRow(ctx, `SELECT id,title,description,category,priority,status,requester_id,assignee_id,created_at,updated_at
		FROM tickets WHERE id=$1`, id))
}

func (p *Postgres) UpdateTicket(ctx context.Context, t domain.Ticket) (domain.Ticket, error) {
	return scanTicket(p.DB.QueryRow(ctx, `UPDATE tickets SET title=$2,description=$3,category=$4,priority=$5,
		status=$6,assignee_id=$7,updated_at=NOW() WHERE id=$1
		RETURNING id,title,description,category,priority,status,requester_id,assignee_id,created_at,updated_at`,
		t.ID, t.Title, t.Description, t.Category, t.Priority, t.Status, t.AssigneeID))
}

func (p *Postgres) DeleteTicket(ctx context.Context, id int64, actor domain.User) (bool, error) {
	tag, err := p.DB.Exec(ctx, `DELETE FROM tickets WHERE id=$1 AND
		($2='admin' OR ($2='requester' AND requester_id=$3 AND status='open'))`,
		id, actor.Role, actor.ID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (p *Postgres) Comments(ctx context.Context, ticketID int64, includeInternal bool) ([]domain.Comment, error) {
	rows, err := p.DB.Query(ctx, `SELECT c.id,c.ticket_id,c.author_id,u.name,c.body,c.is_internal,c.created_at
		FROM comments c JOIN users u ON u.id=c.author_id
		WHERE c.ticket_id=$1 AND ($2::boolean OR c.is_internal=false) ORDER BY c.created_at,c.id`, ticketID, includeInternal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.AuthorName, &c.Body, &c.IsInternal, &c.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (p *Postgres) CreateComment(ctx context.Context, ticketID, authorID int64, body string, internal bool) (domain.Comment, error) {
	var c domain.Comment
	err := p.DB.QueryRow(ctx, `INSERT INTO comments(ticket_id,author_id,body,is_internal)
		VALUES($1,$2,$3,$4) RETURNING id,ticket_id,author_id,body,is_internal,created_at`,
		ticketID, authorID, body, internal).
		Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.IsInternal, &c.CreatedAt)
	if err != nil {
		return c, err
	}
	return scanComment(p.DB.QueryRow(ctx, `SELECT c.id,c.ticket_id,c.author_id,u.name,c.body,c.is_internal,c.created_at
		FROM comments c JOIN users u ON u.id=c.author_id WHERE c.id=$1`, c.ID))
}

func scanUser(row interface{ Scan(...any) error }) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	return u, err
}
func scanTicket(row interface{ Scan(...any) error }) (domain.Ticket, error) {
	var t domain.Ticket
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Category, &t.Priority, &t.Status,
		&t.RequesterID, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}
func scanComment(row interface{ Scan(...any) error }) (domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.AuthorName, &c.Body, &c.IsInternal, &c.CreatedAt)
	return c, err
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
func IsConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
