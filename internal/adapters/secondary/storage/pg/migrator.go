package pg

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"log/slog"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations применяет миграции к базе данных
func RunMigrations(db *sqlx.DB, logger *slog.Logger) error {
	logger.Info("starting database migrations")

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Получаем список миграций
	migrations, err := getMigrations()
	if err != nil {
		return fmt.Errorf("failed to get migrations: %w", err)
	}

	// Получаем текущую версию БД
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Применяем миграции по порядку
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			logger.Debug("migration already applied", "version", migration.Version, "name", migration.Name)
			continue
		}

		logger.Info("applying migration", "version", migration.Version, "name", migration.Name)

		if err := applyMigration(db, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		if err := recordMigration(db, migration.Version); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		logger.Info("migration applied successfully", "version", migration.Version, "name", migration.Name)
	}

	logger.Info("database migrations completed", "applied", len(migrations))
	return nil
}

type migration struct {
	Version int64
	Name    string
	Content string
}

// getMigrations читает все SQL файлы из директории migrations и сортирует их по версии
func getMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []migration

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Парсим версию из имени файла (формат: 0001_name.sql)
		version, name, err := parseMigrationName(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("invalid migration name %s: %w", entry.Name(), err)
		}

		// Читаем содержимое файла
		content, err := migrationsFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration{
			Version: version,
			Name:    name,
			Content: string(content),
		})
	}

	// Сортируем по версии
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationName парсит имя файла миграции (формат: 0001_name.sql)
func parseMigrationName(filename string) (int64, string, error) {
	// Убираем расширение .sql
	name := strings.TrimSuffix(filename, ".sql")

	// Разделяем по первому подчеркиванию
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid format: expected NNNN_name.sql")
	}

	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid version number: %w", err)
	}

	return version, parts[1], nil
}

// applyMigration применяет миграцию к БД
func applyMigration(db *sqlx.DB, m migration) error {
	// Выполняем миграцию в транзакции
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Выполняем SQL
	if _, err := tx.Exec(m.Content); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Отмечаем как dirty на случай ошибки
	if err := markDirtyTx(tx, m.Version, true); err != nil {
		return fmt.Errorf("failed to mark dirty: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Убираем dirty флаг после успешного выполнения
	if err := markDirty(db, m.Version, false); err != nil {
		return fmt.Errorf("failed to unmark dirty: %w", err)
	}

	return nil
}

// getCurrentVersion получает текущую версию БД
func getCurrentVersion(db *sqlx.DB) (int64, error) {
	var version int64
	err := db.Get(&version, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE dirty = false")
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, nil
}

// recordMigration записывает информацию о выполненной миграции
func recordMigration(db *sqlx.DB, version int64) error {
	query := `
		INSERT INTO schema_migrations (version, dirty, applied_at)
		VALUES ($1, false, NOW())
		ON CONFLICT (version) DO UPDATE SET dirty = false, applied_at = NOW()
	`
	_, err := db.Exec(query, version)
	return err
}

// markDirty устанавливает флаг dirty для миграции
func markDirty(db *sqlx.DB, version int64, dirty bool) error {
	query := `
		INSERT INTO schema_migrations (version, dirty, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (version) DO UPDATE SET dirty = $2
	`
	_, err := db.Exec(query, version, dirty)
	return err
}

// markDirtyTx устанавливает флаг dirty для миграции в транзакции
func markDirtyTx(tx *sqlx.Tx, version int64, dirty bool) error {
	query := `
		INSERT INTO schema_migrations (version, dirty, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (version) DO UPDATE SET dirty = $2
	`
	_, err := tx.Exec(query, version, dirty)
	return err
}

// createMigrationsTable создает таблицу для отслеживания выполненных миграций
func createMigrationsTable(db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT NOT NULL PRIMARY KEY,
			dirty BOOLEAN NOT NULL DEFAULT FALSE,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}
