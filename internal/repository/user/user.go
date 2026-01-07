package userRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

type userColumns struct {
	TableName               string
	ID                      string
	TelegramUserID          string
	TelegramChatID          string
	FirstName               string
	LastName                string
	Username                string
	BirthDateTime           string
	BirthPlace              string
	BirthDataSetAt          string
	BirthDataCanChangeUntil string
	NatalChart              string
	NatalChartFetchedAt     string
	CreatedAt               string
	UpdatedAt               string
	LastSeenAt              string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns userColumns
}

// New создаёт новый репозиторий для работы с пользователями
func New(db persistence.Persistence, log *slog.Logger) ports.IUserRepo {
	cols := userColumns{
		TableName:               "tg_users",
		ID:                      "id",
		TelegramUserID:          "tg_id",
		TelegramChatID:          "chat_id",
		FirstName:               "first_name",
		LastName:                "last_name",
		Username:                "username",
		BirthDateTime:           "birth_datetime",
		BirthPlace:              "birth_place",
		BirthDataSetAt:          "birth_data_set_at",
		BirthDataCanChangeUntil: "birth_data_can_change_until",
		NatalChart:              "natal_chart",
		NatalChartFetchedAt:     "natal_chart_fetched_at",
		CreatedAt:               "created_at",
		UpdatedAt:               "updated_at",
		LastSeenAt:              "last_seen_at",
	}
	return &Repository{
		db:      db,
		Log:     log,
		columns: cols,
	}
}

// allColumns возвращает строку со всеми колонками (15 колонок)
func (r *Repository) allColumns() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s",
		r.columns.ID,
		r.columns.TelegramUserID,
		r.columns.TelegramChatID,
		r.columns.FirstName,
		r.columns.LastName,
		r.columns.Username,
		r.columns.BirthDateTime,
		r.columns.BirthPlace,
		r.columns.BirthDataSetAt,
		r.columns.BirthDataCanChangeUntil,
		r.columns.NatalChart,
		r.columns.NatalChartFetchedAt,
		r.columns.CreatedAt,
		r.columns.UpdatedAt,
		r.columns.LastSeenAt)
}

// allColumnsExceptNatalChart возвращает строку со всеми колонками кроме natal_chart (14 колонок)
func (r *Repository) allColumnsExceptNatalChart() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s",
		r.columns.ID,
		r.columns.TelegramUserID,
		r.columns.TelegramChatID,
		r.columns.FirstName,
		r.columns.LastName,
		r.columns.Username,
		r.columns.BirthDateTime,
		r.columns.BirthPlace,
		r.columns.BirthDataSetAt,
		r.columns.BirthDataCanChangeUntil,
		r.columns.NatalChartFetchedAt,
		r.columns.CreatedAt,
		r.columns.UpdatedAt,
		r.columns.LastSeenAt)
}

// Create создаёт нового пользователя
func (r *Repository) Create(ctx context.Context, user *domain.User) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		r.columns.TableName,
		r.allColumns())
	err := r.db.Exec(ctx, query,
		user.ID,
		user.TelegramUserID,
		user.TelegramChatID,
		user.FirstName,
		user.LastName,
		user.Username,
		user.BirthDateTime,
		user.BirthPlace,
		user.BirthDataSetAt,
		user.BirthDataCanChangeUntil,
		user.NatalChart,
		user.NatalChartFetchedAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastSeenAt)
	if err != nil {
		r.Log.Error("failed to create user",
			"error", err,
			"telegram_user_id", user.TelegramUserID,
			"user_id", user.ID)
		return fmt.Errorf("failed to create user: %w", err)
	}
	r.Log.Debug("user created successfully",
		"id", user.ID,
		"telegram_user_id", user.TelegramUserID)
	return nil
}

// GetByTelegramID получает пользователя по Telegram ID (без natal_chart для ленивой загрузки)
func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	var user domain.User
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumnsExceptNatalChart(),
		r.columns.TableName,
		r.columns.TelegramUserID)
	r.Log.Debug("executing query", "query", query, "telegram_id", telegramID)
	err := r.db.Get(ctx, &user, query, telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("user not found", "telegram_user_id", telegramID)
			return nil, fmt.Errorf("user not found: %w", err)
		}
		r.Log.Error("failed to get user by telegram id",
			"error", err,
			"telegram_user_id", telegramID)
		return nil, fmt.Errorf("failed to get user by telegram id: %w", err)
	}
	r.Log.Debug("user retrieved successfully", "telegram_user_id", telegramID, "user_id", user.ID)
	return &user, nil
}

// GetByID получает пользователя по ID (без natal_chart для ленивой загрузки)
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	// Загружаем все колонки кроме natal_chart для оптимизации
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumnsExceptNatalChart(),
		r.columns.TableName,
		r.columns.ID)
	err := r.db.Get(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("user not found", "user_id", id)
			return nil, fmt.Errorf("user not found: %w", err)
		}
		r.Log.Error("failed to get user by id",
			"error", err,
			"user_id", id)
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	r.Log.Debug("user retrieved successfully", "user_id", id)
	return &user, nil
}

// Update обновляет пользователя
func (r *Repository) Update(ctx context.Context, user *domain.User) error {
	query := fmt.Sprintf(`UPDATE %s SET 
		%s = $2, %s = $3, %s = $4, %s = $5, %s = $6, 
		%s = $7, %s = $8, %s = $9, %s = $10, %s = $11, 
		%s = $12, %s = $13, %s = $14
		WHERE %s = $1`,
		r.columns.TableName,
		r.columns.TelegramUserID,
		r.columns.TelegramChatID,
		r.columns.FirstName,
		r.columns.LastName,
		r.columns.Username,
		r.columns.BirthDateTime,
		r.columns.BirthPlace,
		r.columns.BirthDataSetAt,
		r.columns.BirthDataCanChangeUntil,
		r.columns.NatalChart,
		r.columns.NatalChartFetchedAt,
		r.columns.UpdatedAt,
		r.columns.LastSeenAt,
		r.columns.ID)
	rowsAffected, err := r.db.ExecWithResult(ctx, query,
		user.ID,
		user.TelegramUserID,
		user.TelegramChatID,
		user.FirstName,
		user.LastName,
		user.Username,
		user.BirthDateTime,
		user.BirthPlace,
		user.BirthDataSetAt,
		user.BirthDataCanChangeUntil,
		user.NatalChart,
		user.NatalChartFetchedAt,
		user.UpdatedAt,
		user.LastSeenAt)
	if err != nil {
		r.Log.Error("failed to update user",
			"error", err,
			"user_id", user.ID)
		return fmt.Errorf("failed to update user: %w", err)
	}
	if rowsAffected == 0 {
		r.Log.Warn("user not found for update", "user_id", user.ID)
		return fmt.Errorf("user not found")
	}
	r.Log.Debug("user updated successfully", "user_id", user.ID, "rowsAffected", rowsAffected)
	return nil
}

// UpdateLastSeen обновляет время последней активности пользователя
func (r *Repository) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	query := fmt.Sprintf(`UPDATE %s SET %s = $1, %s = $2 WHERE %s = $3`,
		r.columns.TableName,
		r.columns.LastSeenAt,
		r.columns.UpdatedAt,
		r.columns.ID)
	rowsAffected, err := r.db.ExecWithResult(ctx, query, now, now, userID)
	if err != nil {
		r.Log.Error("failed to update last seen",
			"error", err,
			"user_id", userID)
		return fmt.Errorf("failed to update last seen: %w", err)
	}
	if rowsAffected == 0 {
		r.Log.Warn("user not found for update last seen", "user_id", userID)
		return fmt.Errorf("user not found")
	}
	r.Log.Debug("last seen updated successfully", "user_id", userID)
	return nil
}

// BeginTx явно начинает транзакцию
func (r *Repository) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return r.db.BeginTx(ctx)
}

// WithTransaction выполняет функцию в транзакции с автоматическим commit/rollback
func (r *Repository) WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error {
	return r.db.WithTransaction(ctx, fn)
}

// CreateTx создаёт пользователя в транзакции
func (r *Repository) CreateTx(ctx context.Context, tx persistence.Transaction, user *domain.User) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		r.columns.TableName,
		r.allColumns())
	err := tx.Exec(ctx, query,
		user.ID,
		user.TelegramUserID,
		user.TelegramChatID,
		user.FirstName,
		user.LastName,
		user.Username,
		user.BirthDateTime,
		user.BirthPlace,
		user.BirthDataSetAt,
		user.BirthDataCanChangeUntil,
		user.NatalChart,
		user.NatalChartFetchedAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastSeenAt)
	if err != nil {
		r.Log.Error("failed to create user in transaction",
			"error", err,
			"telegram_user_id", user.TelegramUserID,
			"user_id", user.ID)
		return fmt.Errorf("failed to create user in transaction: %w", err)
	}
	r.Log.Debug("user created in transaction",
		"id", user.ID,
		"telegram_user_id", user.TelegramUserID)
	return nil
}

// UpdateTx обновляет пользователя в транзакции
func (r *Repository) UpdateTx(ctx context.Context, tx persistence.Transaction, user *domain.User) error {
	query := fmt.Sprintf(`UPDATE %s SET 
		%s = $2, %s = $3, %s = $4, %s = $5, %s = $6, 
		%s = $7, %s = $8, %s = $9, %s = $10, %s = $11, 
		%s = $12, %s = $13, %s = $14
		WHERE %s = $1`,
		r.columns.TableName,
		r.columns.TelegramUserID,
		r.columns.TelegramChatID,
		r.columns.FirstName,
		r.columns.LastName,
		r.columns.Username,
		r.columns.BirthDateTime,
		r.columns.BirthPlace,
		r.columns.BirthDataSetAt,
		r.columns.BirthDataCanChangeUntil,
		r.columns.NatalChart,
		r.columns.NatalChartFetchedAt,
		r.columns.UpdatedAt,
		r.columns.LastSeenAt,
		r.columns.ID)
	rowsAffected, err := tx.ExecWithResult(ctx, query,
		user.ID,
		user.TelegramUserID,
		user.TelegramChatID,
		user.FirstName,
		user.LastName,
		user.Username,
		user.BirthDateTime,
		user.BirthPlace,
		user.BirthDataSetAt,
		user.BirthDataCanChangeUntil,
		user.NatalChart,
		user.NatalChartFetchedAt,
		user.UpdatedAt,
		user.LastSeenAt)
	if err != nil {
		r.Log.Error("failed to update user in transaction",
			"error", err,
			"user_id", user.ID)
		return fmt.Errorf("failed to update user in transaction: %w", err)
	}
	if rowsAffected == 0 {
		r.Log.Warn("user not found for update in transaction", "user_id", user.ID)
		return fmt.Errorf("user not found")
	}
	r.Log.Debug("user updated in transaction", "user_id", user.ID, "rowsAffected", rowsAffected)
	return nil
}

// GetNatalChart получает только натальную карту/отчёт пользователя (ленивая загрузка)
// Возвращает NatalReport (совместимо с NatalChart)
func (r *Repository) GetNatalChart(ctx context.Context, userID uuid.UUID) (domain.NatalReport, error) {
	var natalChart sql.NullString
	query := fmt.Sprintf(`SELECT COALESCE(%s::text, '') FROM %s WHERE %s = $1`,
		r.columns.NatalChart,
		r.columns.TableName,
		r.columns.ID)
	err := r.db.Get(ctx, &natalChart, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("user not found for natal chart", "user_id", userID)
			return nil, fmt.Errorf("user not found: %w", err)
		}
		r.Log.Error("failed to get natal chart",
			"error", err,
			"user_id", userID)
		return nil, fmt.Errorf("failed to get natal chart: %w", err)
	}
	if !natalChart.Valid || natalChart.String == "" {
		r.Log.Debug("natal chart is empty or null", "user_id", userID)
		return nil, nil
	}
	result := domain.NatalReport(natalChart.String)
	r.Log.Debug("natal chart/report retrieved successfully", "user_id", userID, "size", len(result))
	return result, nil
}

// GetByTelegramIDTx получает пользователя по Telegram ID в транзакции (без natal_chart для ленивой загрузки)
func (r *Repository) GetByTelegramIDTx(ctx context.Context, tx persistence.Transaction, telegramID int64) (*domain.User, error) {
	var user domain.User
	// Загружаем все колонки кроме natal_chart для оптимизации
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumnsExceptNatalChart(),
		r.columns.TableName,
		r.columns.TelegramUserID)
	err := tx.Get(ctx, &user, query, telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("user not found in transaction", "telegram_user_id", telegramID)
			return nil, fmt.Errorf("user not found: %w", err)
		}
		r.Log.Error("failed to get user by telegram id in transaction",
			"error", err,
			"telegram_user_id", telegramID)
		return nil, fmt.Errorf("failed to get user by telegram id in transaction: %w", err)
	}
	r.Log.Debug("user retrieved in transaction", "telegram_user_id", telegramID, "user_id", user.ID)
	return &user, nil
}
