package hizima

import (
	"bytes"
	"strconv"
	"time"
)

// flexTime aceita timestamps devolvidos pela API Hizima em qualquer um dos
// formatos observados na prática, que divergem do que a documentação
// descreve (ela assume string "2006-01-02 15:04:05"; a API real devolve,
// pelo menos para batteryTime, um número epoch em milissegundos).
// Nunca fabrica um valor: se não conseguir parsear, fica nil.
type flexTime struct {
	t *time.Time
}

func (f *flexTime) Time() *time.Time {
	if f == nil {
		return nil
	}
	return f.t
}

func (f *flexTime) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" || string(data) == `""` {
		f.t = nil
		return nil
	}

	// Caso 1: número — epoch em milissegundos (ou segundos, detetado pelo
	// número de dígitos: >= 13 dígitos é milissegundos).
	if data[0] != '"' {
		n, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			f.t = nil
			return nil // nunca fabricar valor — antes nil que um timestamp errado
		}
		if n == 0 {
			f.t = nil
			return nil
		}
		var parsed time.Time
		if n >= 1_000_000_000_000 { // 13+ dígitos -> milissegundos
			parsed = time.UnixMilli(n).UTC()
		} else { // segundos
			parsed = time.Unix(n, 0).UTC()
		}
		f.t = &parsed
		return nil
	}

	// Caso 2: string — tentar RFC3339 primeiro (campos *Utc/*Iso), depois
	// o formato local "2006-01-02 15:04:05" documentado.
	s := string(bytes.Trim(data, `"`))
	if s == "" {
		f.t = nil
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		f.t = &parsed
		return nil
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		f.t = &parsed
		return nil
	}
	// Formato desconhecido: não fabricar, ficar nil.
	f.t = nil
	return nil
}