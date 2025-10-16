package forms

import "github.com/ubaniak/qail/internal/config"

func DisplayConfig(cfg config.Config) {
	headers := []string{"Name", "Value"}
	var rows [][]string
	rows = append(rows, []string{"root", cfg.Root})
	rows = append(rows, []string{"editor", cfg.Editor})
	displayTable(headers, rows)
}
