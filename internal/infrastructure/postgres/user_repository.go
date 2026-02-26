package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.UserRepository = (*UserRepo)(nil)

// UserRepo implementación del puerto UserRepository sobre PostgreSQL.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepository construye el adaptador de persistencia para usuarios.
func NewUserRepository(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create persiste un nuevo usuario.
func (r *UserRepo) Create(user *entity.User) error {
	query := `
		INSERT INTO users (id, company_id, email, password_hash, name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(context.Background(), query,
		user.ID, user.CompanyID, user.Email, user.PasswordHash, user.Name, user.Role, user.Status,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByID obtiene un usuario por ID.
func (r *UserRepo) GetByID(id string) (*entity.User, error) {
	return r.findByID(context.Background(), id)
}

// FindByID alias para GetByID.
func (r *UserRepo) FindByID(id string) (*entity.User, error) {
	return r.GetByID(id)
}

// GetByEmail obtiene un usuario por email (cualquier company).
func (r *UserRepo) GetByEmail(email string) (*entity.User, error) {
	return r.findByEmail(context.Background(), email)
}

// FindByEmail alias para GetByEmail.
func (r *UserRepo) FindByEmail(email string) (*entity.User, error) {
	return r.GetByEmail(email)
}

// GetByEmailAndCompany obtiene un usuario por email y company.
func (r *UserRepo) GetByEmailAndCompany(email, companyID string) (*entity.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, status, created_at, updated_at
		FROM users WHERE email = $1 AND company_id = $2`
	var u entity.User
	err := r.pool.QueryRow(context.Background(), query, email, companyID).Scan(
		&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email and company: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) findByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, status, created_at, updated_at
		FROM users WHERE id = $1`
	var u entity.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) findByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, status, created_at, updated_at
		FROM users WHERE email = $1 LIMIT 1`
	var u entity.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

// Update actualiza un usuario.
func (r *UserRepo) Update(user *entity.User) error {
	query := `
		UPDATE users SET email = $2, password_hash = $3, name = $4, role = $5, status = $6, updated_at = $7
		WHERE id = $1`
	_, err := r.pool.Exec(context.Background(), query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.Role, user.Status, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// ListByCompany lista usuarios por company con paginación.
func (r *UserRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.User, error) {
	query := `
		SELECT id, company_id, email, password_hash, name, role, status, created_at, updated_at
		FROM users WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(context.Background(), query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var list []*entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		list = append(list, &u)
	}
	return list, rows.Err()
}

// Delete elimina un usuario por ID.
func (r *UserRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
