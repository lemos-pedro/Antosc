package reports

import (
	"os"
	"strings"

	"github.com/antosc/aip/internal/prompts"
)

// Persona identifica o destinatário do relatório e o nível de detalhe.
// Alinhado com prompts.Role para que o mesmo perfil no assistente e no
// relatório fale a mesma "língua".
type Persona string

const (
	PersonaOM          Persona = "om"
	PersonaEngenharia  Persona = "engenharia"
	PersonaController  Persona = "controller"
	PersonaFinanceiro  Persona = "financeiro"
	PersonaDiretorTec  Persona = "diretor_tecnico"
	PersonaCEO         Persona = "ceo"
	PersonaConselho    Persona = "conselho_administracao"
)

// AllPersonas é a lista canónica usada pelo job de envio.
var AllPersonas = []Persona{
	PersonaOM,
	PersonaEngenharia,
	PersonaController,
	PersonaFinanceiro,
	PersonaDiretorTec,
	PersonaCEO,
	PersonaConselho,
}

// PersonaMeta descreve como o relatório e o email devem ser construídos.
type PersonaMeta struct {
	ID          Persona
	DisplayName string
	// Foco em texto livre — usado no HTML e no subject.
	Focus string
	// IncludeDetail controla se o Excel leva folhas técnicas completas
	// (O&M, Engenharia) ou só resumo executivo (CEO, Conselho).
	IncludeDetail bool
	// IncludeFinancial inclui desvios de consumo / impacto custo.
	IncludeFinancial bool
	// IncludeActions inclui checklist de ações prioritárias.
	IncludeActions bool
	// IncludeRisk inclui ranking de risco (previsões IA) no HTML/Excel.
	IncludeRisk bool
	// MaxIncidentsInHTML limita linhas na tabela do email (CEO/Conselho).
	MaxIncidentsInHTML int
	// MaxRiskInHTML limita sites de risco no email.
	MaxRiskInHTML int
}

var personaCatalog = map[Persona]PersonaMeta{
	PersonaOM: {
		ID:                 PersonaOM,
		DisplayName:        "O&M",
		Focus:              "Ação imediata: sites prioritários, causa provável e próximo passo",
		IncludeDetail:      true,
		IncludeFinancial:   false,
		IncludeActions:     true,
		IncludeRisk:        true,
		MaxIncidentsInHTML: 50,
		MaxRiskInHTML:      15,
	},
	PersonaEngenharia: {
		ID:                 PersonaEngenharia,
		DisplayName:        "Engenharia",
		Focus:              "Causa-raiz técnica, padrões e histórico de anomalias",
		IncludeDetail:      true,
		IncludeFinancial:   false,
		IncludeActions:     true,
		IncludeRisk:        true,
		MaxIncidentsInHTML: 50,
		MaxRiskInHTML:      20,
	},
	PersonaController: {
		ID:                 PersonaController,
		DisplayName:        "Controller",
		Focus:              "Conformidade: desvios face às normas definidas",
		IncludeDetail:      true,
		IncludeFinancial:   true,
		IncludeActions:     false,
		IncludeRisk:        false,
		MaxIncidentsInHTML: 30,
		MaxRiskInHTML:      0,
	},
	PersonaFinanceiro: {
		ID:                 PersonaFinanceiro,
		DisplayName:        "Financeiro",
		Focus:              "Impacto de custo e desvios de consumo vs norma",
		IncludeDetail:      false,
		IncludeFinancial:   true,
		IncludeActions:     false,
		IncludeRisk:        false,
		MaxIncidentsInHTML: 20,
		MaxRiskInHTML:      0,
	},
	PersonaDiretorTec: {
		ID:                 PersonaDiretorTec,
		DisplayName:        "Diretor Técnico",
		Focus:              "Prioridades cross-site, tendências e decisões de intervenção",
		IncludeDetail:      true,
		IncludeFinancial:   true,
		IncludeActions:     true,
		IncludeRisk:        true,
		MaxIncidentsInHTML: 25,
		MaxRiskInHTML:      15,
	},
	PersonaCEO: {
		ID:                 PersonaCEO,
		DisplayName:        "CEO",
		Focus:              "Risco de negócio, tendência geral e impacto operacional",
		IncludeDetail:      false,
		IncludeFinancial:   true,
		IncludeActions:     false,
		IncludeRisk:        true,
		MaxIncidentsInHTML: 8,
		MaxRiskInHTML:      5,
	},
	PersonaConselho: {
		ID:                 PersonaConselho,
		DisplayName:        "Conselho de Administração",
		Focus:              "KPIs agregados e risco a nível de portfolio",
		IncludeDetail:      false,
		IncludeFinancial:   true,
		IncludeActions:     false,
		IncludeRisk:        true,
		MaxIncidentsInHTML: 5,
		MaxRiskInHTML:      3,
	},
}

func MetaFor(p Persona) PersonaMeta {
	if m, ok := personaCatalog[p]; ok {
		return m
	}
	return PersonaMeta{
		ID:                 p,
		DisplayName:        string(p),
		Focus:              "Resumo operacional",
		IncludeDetail:      true,
		MaxIncidentsInHTML: 20,
	}
}

// RoleString mapeia persona → role do assistente (prompts package).
func (p Persona) RoleString() string {
	return string(p)
}

// SystemPrompt devolve o prompt de sistema alinhado com o assistente.
func (p Persona) SystemPrompt() string {
	return prompts.SystemPromptFor(string(p))
}

// RecipientsFromEnv lê destinatários por persona.
// Formato: REPORT_RECIPIENTS_OM=a@x.com,b@y.com
//          REPORT_RECIPIENTS_CEO=ceo@empresa.com
// Se a variável da persona estiver vazia, o envio para essa persona é omitido.
func RecipientsFromEnv(p Persona) []string {
	key := "REPORT_RECIPIENTS_" + strings.ToUpper(string(p))
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if e := strings.TrimSpace(part); e != "" {
			out = append(out, e)
		}
	}
	return out
}
