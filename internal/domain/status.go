package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RequestStatus int16

const (
	RequestSentToRAG RequestStatus = 1  // отправлен в RAG (финальный статус этапа отправки)
	RequestCompleted RequestStatus = 2  // успешно завершён (финальный статус)
	RequestError     RequestStatus = 99 // ошибка на любом этапе
)

// RequestPhase фаза обработки запроса (группировка этапов)
type RequestPhase string

const (
	PhaseSend      RequestPhase = "send"    // этап отправки в Kafka (text_handler.go)
	PhaseReceive   RequestPhase = "receive" // этап приёма из Kafka (rag_response.go)
	PhaseUndefined RequestPhase = "undefined"
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
	StageGetRequest   RequestStage = "get_request"   // получение запроса из БД
	StageSaveResponse RequestStage = "save_response" // сохранение ответа в БД
	StageGetUser      RequestStage = "get_user"      // получение пользователя из БД
	StageSendTelegram RequestStage = "send_telegram" // отправка сообщения в Telegram
)

// GetPhase возвращает фазу для этапа
func (s RequestStage) GetPhase() RequestPhase {
	switch s {
	case StageCreateRequest, StageLoadNatalChart, StageKafkaSend:
		return PhaseSend
	case StageGetRequest, StageSaveResponse, StageGetUser, StageSendTelegram:
		return PhaseReceive
	default:
		return PhaseUndefined
	}
}

type ObjectType string

const (
	ObjectTypeRequest ObjectType = "request"
)

type Status struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ObjectType   ObjectType      `json:"object_type" db:"object_type"`
	ObjectID     uuid.UUID       `json:"object_id" db:"object_id"`
	Status       RequestStatus   `json:"status" db:"status"`
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
