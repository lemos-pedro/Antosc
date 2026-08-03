package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/assistant"
	"github.com/antosc/aip/internal/auth"
	"github.com/antosc/aip/internal/prompts"
)

type AssistantHandler struct {
	service *assistant.Service
	log     *slog.Logger
}

func NewAssistantHandler(service *assistant.Service, log *slog.Logger) *AssistantHandler {
	return &AssistantHandler{service: service, log: log}
}

// Ask — POST /api/v1/assistant/ask
// Body: {"role": "controller", "question": "quais sites caíram esta semana?"}
// A resposta inclui "grounded": false se a verificação anti-alucinação
// detetar números na resposta que não vêm dos dados obtidos pelas
// ferramentas -- não bloqueia a resposta, mas sinaliza para o utilizador
// (e fica registado em assistant_audit_log para revisão).
func (h *AssistantHandler) Ask(w http.ResponseWriter, r *http.Request) {
	var req dto.AssistantAskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		http.Error(w, "question é obrigatório", http.StatusBadRequest)
		return
	}

	// Se autenticado, a role do token prevalece (admin pode impersonar via body).
	if p, ok := auth.PrincipalFrom(r.Context()); ok {
		if !auth.IsAdmin(p.Role) || req.Role == "" {
			req.Role = p.Role
		}
	}

	systemPrompt := prompts.SystemPromptFor(req.Role)

	result, err := h.service.Ask(r.Context(), req.Role, systemPrompt, req.Question)
	if err != nil {
		h.log.Error("assistente falhou ao responder", "err", err, "role", req.Role)
		http.Error(w, "o assistente não conseguiu responder de momento", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, dto.AssistantAskResponse{
		Answer:   result.Answer,
		Grounded: result.Grounded,
		Suspects: result.Suspects,
	})
}
