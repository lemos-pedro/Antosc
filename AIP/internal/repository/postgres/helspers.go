package postgres

import (
	"strconv"
	"strings"
)

// placeholders gera "($1,$2,...,$n)" para from..to (inclusive), usado na
// construção de multi-row INSERT sem concatenar valores diretamente (evita SQL injection).
func placeholders(from, to int) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i := from; i <= to; i++ {
		if i > from {
			sb.WriteString(",")
		}
		sb.WriteString("$")
		sb.WriteString(strconv.Itoa(i))
	}
	sb.WriteString(")")
	return sb.String()
}
