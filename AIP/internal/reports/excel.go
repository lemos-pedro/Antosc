// Package reports gera exportações (Excel) e o conteúdo do relatório semanal.
package reports

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Table é um conjunto de dados tabular simples -- cabeçalhos + linhas --
// suficiente para qualquer exportação pedida ao assistente ou à API
// (incidentes, desvios de consumo, previsões, etc.). Mantém o gerador de
// Excel desacoplado da forma interna de cada dataset.
type Table struct {
	SheetName string
	Headers   []string
	Rows      [][]any
}

// BuildExcel constrói um ficheiro .xlsx em memória a partir de uma ou mais
// tabelas (uma folha por tabela) e devolve os bytes prontos a servir como
// download.
func BuildExcel(tables []Table) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	for i, t := range tables {
		sheet := t.SheetName
		if sheet == "" {
			sheet = fmt.Sprintf("Folha%d", i+1)
		}

		if i == 0 {
			f.SetSheetName("Sheet1", sheet)
		} else {
			if _, err := f.NewSheet(sheet); err != nil {
				return nil, err
			}
		}

		for col, h := range t.Headers {
			cell, err := excelize.CoordinatesToCellName(col+1, 1)
			if err != nil {
				return nil, err
			}
			f.SetCellValue(sheet, cell, h)
		}

		for rowIdx, row := range t.Rows {
			for col, v := range row {
				cell, err := excelize.CoordinatesToCellName(col+1, rowIdx+2)
				if err != nil {
					return nil, err
				}
				f.SetCellValue(sheet, cell, v)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
