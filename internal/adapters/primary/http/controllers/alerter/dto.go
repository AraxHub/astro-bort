package alerter

// RailwayWebhookPayload структура для парсинга Railway webhook
type RailwayWebhookPayload struct {
	Type      string                `json:"type"`
	Details   RailwayDeploymentInfo `json:"details"`
	Resource  RailwayResource       `json:"resource"`
	Severity  string                `json:"severity"`
	Timestamp string                `json:"timestamp"`
}

// RailwayDeploymentInfo информация о деплое
type RailwayDeploymentInfo struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	Status        string `json:"status"`
	Branch        string `json:"branch"`
	CommitHash    string `json:"commitHash"`
	CommitAuthor  string `json:"commitAuthor"`
	CommitMessage string `json:"commitMessage"`
}

// RailwayResource информация о ресурсах
type RailwayResource struct {
	Workspace   RailwayWorkspace   `json:"workspace"`
	Project     RailwayProject     `json:"project"`
	Environment RailwayEnvironment `json:"environment"`
	Service     RailwayService     `json:"service"`
	Deployment  RailwayDeployment  `json:"deployment"`
}

// RailwayWorkspace информация о workspace
type RailwayWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RailwayProject информация о проекте
type RailwayProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RailwayEnvironment информация о окружении
type RailwayEnvironment struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsEphemeral bool  `json:"isEphemeral"`
}

// RailwayService информация о сервисе
type RailwayService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RailwayDeployment информация о деплое
type RailwayDeployment struct {
	ID string `json:"id"`
}

