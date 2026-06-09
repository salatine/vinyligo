package sheets

import (
	"fmt"

	"github.com/xuri/excelize/v2"
	"github.com/salatine/vinyligo/internal/models"
)

type ResumeSheet struct {
	file *excelize.File
}

func NewResumeSheet() *ResumeSheet {
	return &ResumeSheet{file: excelize.NewFile()}
}

func (r *ResumeSheet) CreateResumeSheet(products []*models.Product, path string, titleEditor func(string) string) error {
	sheet := "Sheet1"

	r.file.SetCellValue(sheet, "A1", "Título")
	r.file.SetCellValue(sheet, "B1", "Preço")
	r.file.SetCellValue(sheet, "C1", "Plataformas")

	style, _ := r.file.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Font:      &excelize.Font{Bold: true},
	})
	for _, col := range []string{"A", "B", "C"} {
		r.file.SetCellStyle(sheet, col+"1", col+"1", style)
	}

	for i, product := range products {
		row := i + 2
		r.file.SetCellValue(sheet, fmt.Sprintf("A%d", row), product.Title(titleEditor))
		r.file.SetCellValue(sheet, fmt.Sprintf("B%d", row), product.Price)
		r.file.SetCellValue(sheet, fmt.Sprintf("C%d", row), "ML, Shopify")

		numStyle, _ := r.file.NewStyle(&excelize.Style{NumFmt: 2})
		r.file.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), numStyle)
	}

	totalRow := len(products) + 2
	r.file.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), len(products))
	r.file.SetCellFormula(sheet, fmt.Sprintf("B%d", totalRow), fmt.Sprintf("=SUM(B2:B%d)", len(products)+1))
	r.file.SetCellValue(sheet, fmt.Sprintf("C%d", totalRow), "ML, Shopify")

	cols, _ := r.file.GetCols(sheet)
	for i, col := range cols {
		maxLen := 0
		for _, cell := range col {
			if len(cell) > maxLen {
				maxLen = len(cell)
			}
		}
		if maxLen > 0 {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			r.file.SetColWidth(sheet, colName, colName, float64(maxLen)*1.23)
		}
	}

	return r.file.SaveAs(path)
}
