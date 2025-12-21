# UX Flow - Астрологический бот

## Блок-схема пользовательского опыта

```mermaid
flowchart TD
    Start([Пользователь отправляет /start]) --> CheckBirthDate{Дата рождения<br/>установлена?}
    
    CheckBirthDate -->|Нет| ShowBirthWarning[Показать предупреждение:<br/>⚠️ Дата устанавливается ОДИН РАЗ<br/>Задавай вопросы только от своего лица]
    ShowBirthWarning --> WaitBirthInput[Ожидание ввода даты:<br/>ДД.ММ.ГГГГ]
    WaitBirthInput --> ValidateBirthDate{Валидация<br/>даты}
    
    ValidateBirthDate -->|Неверный формат| ShowError[Показать ошибку:<br/>❌ Неверный формат<br/>Введи ДД.ММ.ГГГГ]
    ShowError --> WaitBirthInput
    
    ValidateBirthDate -->|Верный формат| SaveBirthDate[Сохранить дату рождения<br/>birth_data_set_at = NOW<br/>birth_data_can_change_until = NOW + 24h]
    SaveBirthDate --> ShowBirthSuccess[Показать:<br/>✅ Дата установлена<br/>⚠️ Можно изменить в течение 24ч]
    
    CheckBirthDate -->|Да| CheckNatalChart{Натальная карта<br/>есть?}
    ShowBirthSuccess --> CheckNatalChart
    
    CheckNatalChart -->|Нет| RequestNatalChart[Запрос в астро-API<br/>с датой рождения]
    RequestNatalChart --> SaveNatalChart{Успешно?}
    
    SaveNatalChart -->|Ошибка| ShowNatalError[Показать ошибку:<br/>❌ Не удалось получить<br/>натальную карту]
    ShowNatalError --> EndError([Ошибка])
    
    SaveNatalChart -->|Успешно| SaveNatalChartDB[Сохранить натальную карту<br/>natal_chart = данные<br/>natal_chart_fetched_at = NOW]
    SaveNatalChartDB --> ShowReady[Показать:<br/>✅ Натальная карта получена!<br/>Готов к работе]
    
    CheckNatalChart -->|Да| ShowReady
    
    ShowReady --> WaitUserQuestion[Ожидание вопроса<br/>от пользователя]
    
    WaitUserQuestion --> CreateRequest[Создать запрос:<br/>requests: user_id, request_text<br/>status: 'received']
    
    CreateRequest --> CheckNatalChartAgain{Натальная карта<br/>есть?}
    
    CheckNatalChartAgain -->|Нет| RequestNatalChart
    
    CheckNatalChartAgain -->|Да| SendToKafka[Отправить в Kafka:<br/>request_id, request_text, natal_chart<br/>status: 'sent_to_rag']
    
    SendToKafka --> WaitRAGResponse[Ожидание ответа<br/>из Kafka топика]
    
    WaitRAGResponse --> ReceiveRAGResponse[Получить ответ из Kafka:<br/>request_id, response_text<br/>status: 'rag_response_received']
    
    ReceiveRAGResponse --> FindUserChat[Найти пользователя:<br/>SELECT u.telegram_chat_id<br/>FROM requests r<br/>JOIN users u ON r.user_id = u.id<br/>WHERE r.id = request_id]
    
    FindUserChat --> SendToUser[Отправить ответ пользователю:<br/>chat_id = telegram_chat_id<br/>status: 'sent_to_user']
    
    SendToUser --> WaitUserQuestion
    
    WaitUserQuestion -->|Команда /reset_birth_data| CheckResetTime{Можно изменить?<br/>birth_data_can_change_until > NOW?}
    
    CheckResetTime -->|Нет| ShowResetError[Показать:<br/>❌ Дата заблокирована<br/>Обратись к администратору]
    ShowResetError --> WaitUserQuestion
    
    CheckResetTime -->|Да| ShowResetWarning[Показать:<br/>⚠️ Ты уверен?<br/>Это удалит дату и натальную карту<br/>Введи 'ПОДТВЕРДИТЬ']
    ShowResetWarning --> WaitConfirm[Ожидание подтверждения]
    
    WaitConfirm -->|'ПОДТВЕРДИТЬ'| ResetBirthData[Сбросить:<br/>birth_date = NULL<br/>natal_chart = NULL<br/>birth_data_set_at = NULL]
    ResetBirthData --> WaitBirthInput
    
    WaitConfirm -->|Другое| WaitUserQuestion
    
    WaitUserQuestion -->|Команда /help| ShowHelp[Показать справку:<br/>/start - Начать<br/>/reset_birth_data - Сбросить дату<br/>/my_info - Моя информация]
    ShowHelp --> WaitUserQuestion
    
    WaitUserQuestion -->|Команда /my_info| ShowUserInfo[Показать:<br/>Дата рождения: birth_date<br/>Натальная карта: ✅/❌]
    ShowUserInfo --> WaitUserQuestion
    
    style Start fill:#90EE90
    style EndError fill:#FFB6C1
    style ShowReady fill:#87CEEB
    style SaveBirthDate fill:#FFD700
    style SaveNatalChartDB fill:#FFD700
    style SendToKafka fill:#DDA0DD
    style ReceiveRAGResponse fill:#98FB98
```

## Описание этапов

### 1. Инициализация
- Пользователь отправляет `/start`
- Проверяется наличие даты рождения

### 2. Установка даты рождения
- Если даты нет → запрос с предупреждением
- Валидация формата (ДД.ММ.ГГГГ)
- Сохранение с ограничением на изменение (24 часа)

### 3. Получение натальной карты
- Проверка наличия натальной карты
- Если нет → запрос в астро-API
- Сохранение результата

### 4. Основной режим работы
- Ожидание вопросов от пользователя
- Создание запроса в БД
- Отправка в Kafka для RAG
- Получение ответа из Kafka
- Отправка ответа пользователю

### 5. Дополнительные команды
- `/reset_birth_data` - сброс даты (только в течение 24 часов)
- `/help` - справка
- `/my_info` - информация о пользователе

