package dto

// AssistantAskRequest é o payload enviado pelo utilizador (via app/frontend)
// ao fazer uma pergunta livre ao assistente.
type AssistantAskRequest struct {
	Role     string `json:"role"`     // ex: "controller", "financeiro", "ceo", "om"...
	Question string `json:"question"`
}

type AssistantAskResponse struct {
	Answer   string   `json:"answer"`
	Grounded bool     `json:"grounded"`
	Suspects []string `json:"suspects,omitempty"`
}
