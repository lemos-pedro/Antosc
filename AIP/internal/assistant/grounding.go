package assistant

import (
	"regexp"
	"strings"
)

// numberPattern captura números (inteiros ou decimais, com ou sem sinal) em texto livre.
var numberPattern = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)

// CheckGrounding verifica se os valores numéricos que o modelo citou na
// resposta final aparecem também nos resultados brutos das ferramentas que
// ele chamou. É uma verificação heurística, não uma prova formal: números
// pequenos e comuns (0, 1, 2...) são ignorados para evitar falsos positivos
// constantes, e não entende se o modelo fez uma soma/média legítima a partir
// de vários números reais. O objetivo não é bloquear a resposta, é sinalizar
// para revisão humana quando algo foge claramente dos dados.
func CheckGrounding(finalAnswer string, toolResults []string) (grounded bool, suspects []string) {
	source := strings.Join(toolResults, " ")
	sourceNumbers := toSet(numberPattern.FindAllString(source, -1))

	answerNumbers := numberPattern.FindAllString(finalAnswer, -1)

	for _, n := range answerNumbers {
		if isIgnorable(n) {
			continue
		}
		if !sourceNumbers[normalize(n)] {
			suspects = append(suspects, n)
		}
	}

	return len(suspects) == 0, suspects
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, it := range items {
		set[normalize(it)] = true
	}
	return set
}

func normalize(n string) string {
	return strings.ReplaceAll(n, ",", ".")
}

// isIgnorable evita sinalizar números triviais que aparecem naturalmente em
// frases (contagens pequenas, percentagens redondas de exemplo, etc.) e que
// geram ruído sem valor de deteção real.
func isIgnorable(n string) bool {
	switch normalize(n) {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "100":
		return true
	default:
		return false
	}
}
