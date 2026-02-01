package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RequestPhase фаза обработки запроса (группировка этапов)
type RequestPhase string

const (
	PhaseSend             RequestPhase = "send"    // этап отправки в Kafka (text_handler.go)
	PhaseReceive          RequestPhase = "receive" // этап приёма из Kafka (rag_response.go)
	RequestPhaseUndefined RequestPhase = "undefined"
)

// RequestStage детальный этап обработки запроса (для ошибок)
type RequestStage string

// Этапы фазы отправки (PhaseSend)
const (
	StageCreateRequest  RequestStage = "create_request"   // создание запроса в БД
	StageLoadNatalChart RequestStage = "load_natal_chart" // загрузка натальной карты
	StageKafkaSend      RequestStage = "kafka_send"       // отправка в Kafka
)

// Этапы фазы приёма (PhaseReceive)
const (
	StageGetRequest      RequestStage = "get_request"       // получение запроса из БД
	StageSaveResponse    RequestStage = "save_response"     // сохранение ответа в БД
	StageGetUser         RequestStage = "get_user"           // получение пользователя из БД
	StageSendTelegram    RequestStage = "send_telegram"      // отправка сообщения в Telegram
	StageGetImageUsage   RequestStage = "get_image_usage"   // получение статистики использования картинок
	StageGetImages       RequestStage = "get_images"         // получение картинок по теме
	StageSelectImage     RequestStage = "select_image"       // выбор картинки по алгоритму
	StageSendPhoto       RequestStage = "send_photo"          // отправка фото в Telegram
	StageIncrementUsage  RequestStage = "increment_usage"     // инкремент счётчика использования
)

// GetPhase возвращает фазу для этапа
func (s RequestStage) GetPhase() RequestPhase {
	switch s {
	case StageCreateRequest, StageLoadNatalChart, StageKafkaSend:
		return PhaseSend
	case StageGetRequest, StageSaveResponse, StageGetUser, StageSendTelegram,
		StageGetImageUsage, StageGetImages, StageSelectImage, StageSendPhoto, StageIncrementUsage:
		return PhaseReceive
	default:
		return RequestPhaseUndefined
	}
}

type ObjectType string

const (
	ObjectTypeRequest ObjectType = "request"
	ObjectTypePayment ObjectType = "payment"
)

// StatusStatus объединяет RequestStatus и PaymentStatus для универсальной таблицы statuses
// В БД хранится как SMALLINT, интерпретация зависит от object_type
type StatusStatus int16

// RequestStatus статусы для запросов
type RequestStatus StatusStatus

const (
	RequestSentToRAG RequestStatus = 1  // отправлен в RAG (финальный статус этапа отправки)
	RequestCompleted RequestStatus = 2  // успешно завершён (финальный статус)
	RequestError     RequestStatus = 99 // ошибка на любом этапе
)

// PaymentStatusEnum статусы для платежей в таблице statuses (отличается от PaymentStatus в payment.go)
type PaymentStatusEnum StatusStatus

const (
	PaymentCreated   PaymentStatusEnum = 1  // создан, invoice отправлен
	PaymentSucceeded PaymentStatusEnum = 2  // успешно оплачен, продукт выдан
	PaymentFailed    PaymentStatusEnum = 3  // оплата не прошла (invoice не создан, отклонён)
	PaymentError     PaymentStatusEnum = 99 // критическая ошибка (деньги списаны, но продукт не выдан)
)

type Status struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ObjectType   ObjectType      `json:"object_type" db:"object_type"`
	ObjectID     uuid.UUID       `json:"object_id" db:"object_id"`
	Status       StatusStatus    `json:"status" db:"status"` // универсальный статус (RequestStatus или PaymentStatus)
	ErrorMessage *string         `json:"error_message,omitempty" db:"error_message"`
	Metadata     json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// BuildErrorMetadata создаёт metadata для ошибки
func BuildErrorMetadata(stage RequestStage, errorCode string, botID string, context map[string]interface{}) json.RawMessage {
	phase := stage.GetPhase()

	m := map[string]interface{}{
		"phase":      string(phase),
		"stage":      string(stage),
		"error_code": errorCode,
		"bot_id":     botID,
	}

	if len(context) > 0 {
		m["context"] = context
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}
	return json.RawMessage(data)
}

// BuildKafkaMetadata создаёт metadata для успешной отправки в Kafka
func BuildKafkaMetadata(topic string, partition int32, offset int64, botID string, textLength, natalChartSize int) json.RawMessage {
	m := map[string]interface{}{
		"kafka": map[string]interface{}{
			"topic":     topic,
			"partition": partition,
			"offset":    offset,
		},
		"request": map[string]interface{}{
			"text_length":      textLength,
			"natal_chart_size": natalChartSize,
		},
		"bot_id": botID,
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}
	return json.RawMessage(data)
}

// BuildTelegramMetadata создаёт metadata для успешной отправки в Telegram
func BuildTelegramMetadata(messageID, chatID int64, botID string, responseLength int) json.RawMessage {
	m := map[string]interface{}{
		"telegram": map[string]interface{}{
			"message_id": messageID,
			"chat_id":    chatID,
		},
		"response": map[string]interface{}{
			"length": responseLength,
		},
		"bot_id": botID,
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}
	return json.RawMessage(data)
}

// PaymentPhase фаза обработки платежа
type PaymentPhase string

const (
	PhaseCreate           PaymentPhase = "create"     // создание платежа и invoice
	PhaseValidation       PaymentPhase = "validation" // pre-checkout валидация
	PhaseProcessing       PaymentPhase = "processing" // обработка успешного платежа
	PaymentPhaseUndefined PaymentPhase = "undefined"
)

// PaymentStage детальный этап обработки платежа
type PaymentStage string

// Этапы фазы создания (PhaseCreate)
const (
	StageCreatePayment PaymentStage = "create_payment" // создание записи в БД
	StageCreateInvoice PaymentStage = "create_invoice" // отправка invoice через Telegram API
)

// Этапы фазы валидации (PhaseValidation)
const (
	StagePreCheckoutValidation PaymentStage = "pre_checkout_validation" // валидация pre-checkout
)

// Этапы фазы обработки (PhaseProcessing)
const (
	StageUpdateStatus     PaymentStage = "update_status"     // обновление статуса на succeeded
	StageGrantProduct     PaymentStage = "grant_product"     // выдача продукта (is_paid=true)
	StageSendNotification PaymentStage = "send_notification" // отправка уведомления пользователю
)

// GetPhase возвращает фазу для этапа платежа
func (s PaymentStage) GetPhase() PaymentPhase {
	switch s {
	case StageCreatePayment, StageCreateInvoice:
		return PhaseCreate
	case StagePreCheckoutValidation:
		return PhaseValidation
	case StageUpdateStatus, StageGrantProduct, StageSendNotification:
		return PhaseProcessing
	default:
		return PaymentPhaseUndefined
	}
}

// BuildPaymentErrorMetadata создаёт metadata для ошибки платежа
func BuildPaymentErrorMetadata(stage PaymentStage, errorCode string, botID string, context map[string]interface{}) json.RawMessage {
	phase := stage.GetPhase()

	m := map[string]interface{}{
		"phase":      string(phase),
		"stage":      string(stage),
		"error_code": errorCode,
		"bot_id":     botID,
	}

	if len(context) > 0 {
		m["context"] = context
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}
	return json.RawMessage(data)
}

// BuildPaymentSuccessMetadata создаёт metadata для успешной операции платежа
func BuildPaymentSuccessMetadata(stage PaymentStage, botID string, context map[string]interface{}) json.RawMessage {
	phase := stage.GetPhase()

	m := map[string]interface{}{
		"phase":  string(phase),
		"stage":  string(stage),
		"bot_id": botID,
	}

	if len(context) > 0 {
		m["context"] = context
	}

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}
	return json.RawMessage(data)
}
