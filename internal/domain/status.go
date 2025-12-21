package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RequestStatus int16

const (
	RequestReceived          RequestStatus = 1  // пришёл запрос в сообщении
	RequestSentToRAG         RequestStatus = 2  // отправлен в RAG
	RequestRAGRespReceived   RequestStatus = 3  // получен финальный положительный ответ
	RequestRAGRespSentToUser RequestStatus = 4  // отправлен финальный положительный ответ
	RequestRAGRespError      RequestStatus = 5  // получен финальный отрицательный ответ
	RequestRAGRespErrorSent  RequestStatus = 6  // отправлена финальная ошибка в ответ пользователю
	RequestFailed            RequestStatus = 99 // ошибка
)

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
