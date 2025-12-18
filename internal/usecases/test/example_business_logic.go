package testService

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

// Пример бизнес-логики с транзакциями

// CreateOrderWithItems создаёт заказ с несколькими позициями атомарно
// Если хотя бы одна операция не удалась - все изменения откатываются
func (s *Service) CreateOrderWithItems(ctx context.Context, orderName string, items []*domain.Test) error {
	// Описываем функцию с параметром tx - она будет вызвана внутри WithTransaction
	// tx создастся автоматически и подставится в этот параметр
	fn := func(ctx context.Context, tx persistence.Transaction) error {
		// 1. Создаём главную запись (заказ)
		order := &domain.Test{
			Filed1: orderName,
			Filed2: len(items), // количество позиций
		}

		if err := s.TestRepo.CreateTx(ctx, tx, order); err != nil {
			s.Log.Error("failed to create order in transaction", "error", err, "order_name", orderName)
			return fmt.Errorf("failed to create order: %w", err)
		}

		s.Log.Info("order created in transaction", "order_id", order.ID)

		// 2. Создаём все позиции заказа
		for i, item := range items {
			item.Filed1 = fmt.Sprintf("%s-item-%d", orderName, i+1)
			item.Filed2 = i + 1

			if err := s.TestRepo.CreateTx(ctx, tx, item); err != nil {
				s.Log.Error("failed to create item in transaction",
					"error", err,
					"item_index", i,
					"order_id", order.ID)
				// Если вернём ошибку - транзакция откатится автоматически
				return fmt.Errorf("failed to create item %d: %w", i+1, err)
			}

			s.Log.Debug("item created in transaction", "item_id", item.ID, "order_id", order.ID)
		}

		// 3. Обновляем заказ с итоговой информацией
		order.Filed2 = len(items) * 100 // какая-то бизнес-логика
		if err := s.TestRepo.UpdateTx(ctx, tx, order); err != nil {
			s.Log.Error("failed to update order in transaction", "error", err, "order_id", order.ID)
			return fmt.Errorf("failed to update order: %w", err)
		}

		s.Log.Info("order with items created successfully",
			"order_id", order.ID,
			"items_count", len(items))

		// Если всё ок - транзакция зафиксируется автоматически
		// Если вернём ошибку - транзакция откатится автоматически
		return nil
	}

	// Вызываем WithTransaction - он создаст tx, вызовет fn с этой tx, и зафиксирует/откатит
	if err := s.TestRepo.WithTransaction(ctx, fn); err != nil {
		s.Log.Error("transaction failed", "error", err, "order_name", orderName)
		return fmt.Errorf("failed to create order with items: %w", err)
	}

	return nil
}

// TransferData атомарно переносит данные из одной записи в другую
func (s *Service) TransferData(ctx context.Context, fromID, toID int64) error {
	// Описываем функцию - tx будет создана и подставлена автоматически
	fn := func(ctx context.Context, tx persistence.Transaction) error {
		// 1. Получаем исходную запись
		from, err := s.TestRepo.GetByIDTx(ctx, tx, fromID)
		if err != nil {
			s.Log.Error("failed to get source record", "error", err, "from_id", fromID)
			return fmt.Errorf("failed to get source record: %w", err)
		}

		// 2. Получаем целевую запись
		to, err := s.TestRepo.GetByIDTx(ctx, tx, toID)
		if err != nil {
			s.Log.Error("failed to get target record", "error", err, "to_id", toID)
			return fmt.Errorf("failed to get target record: %w", err)
		}

		// 3. Переносим данные
		to.Filed1 = from.Filed1
		to.Filed2 = from.Filed2

		// 4. Обновляем целевую запись
		if err := s.TestRepo.UpdateTx(ctx, tx, to); err != nil {
			s.Log.Error("failed to update target", "error", err, "to_id", toID)
			return fmt.Errorf("failed to update target: %w", err)
		}

		// 5. Очищаем исходную запись
		from.Filed1 = ""
		from.Filed2 = 0
		if err := s.TestRepo.UpdateTx(ctx, tx, from); err != nil {
			s.Log.Error("failed to clear source", "error", err, "from_id", fromID)
			return fmt.Errorf("failed to clear source: %w", err)
		}

		s.Log.Info("data transferred successfully", "from_id", fromID, "to_id", toID)
		return nil
	}

	// Выполняем в транзакции
	if err := s.TestRepo.WithTransaction(ctx, fn); err != nil {
		s.Log.Error("transaction failed", "error", err, "from_id", fromID, "to_id", toID)
		return fmt.Errorf("failed to transfer data: %w", err)
	}

	return nil
}

// DeleteOrderWithItems удаляет заказ и все его позиции атомарно
func (s *Service) DeleteOrderWithItems(ctx context.Context, orderID int64, itemIDs []int64) error {
	// Описываем функцию - tx будет создана и подставлена автоматически
	fn := func(ctx context.Context, tx persistence.Transaction) error {
		// 1. Удаляем все позиции
		for _, itemID := range itemIDs {
			if err := s.TestRepo.DeleteTx(ctx, tx, itemID); err != nil {
				s.Log.Error("failed to delete item", "error", err, "item_id", itemID, "order_id", orderID)
				return fmt.Errorf("failed to delete item %d: %w", itemID, err)
			}
		}

		// 2. Удаляем заказ
		if err := s.TestRepo.DeleteTx(ctx, tx, orderID); err != nil {
			s.Log.Error("failed to delete order", "error", err, "order_id", orderID)
			return fmt.Errorf("failed to delete order: %w", err)
		}

		s.Log.Info("order and items deleted successfully",
			"order_id", orderID,
			"items_count", len(itemIDs))

		return nil
	}

	// Выполняем в транзакции
	if err := s.TestRepo.WithTransaction(ctx, fn); err != nil {
		s.Log.Error("transaction failed", "error", err, "order_id", orderID)
		return fmt.Errorf("failed to delete order with items: %w", err)
	}

	return nil
}
