package forms

import c "qail/internal/config"

func DisplayConfig(cfg c.Config) {
	headers := []string{"Name", "Value"}
	var rows [][]string
	rows = append(rows, []string{"root", cfg.Root})
	rows = append(rows, []string{"editor", cfg.Editor})
	displayTable(headers, rows)
}
