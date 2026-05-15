package forms

import "github.com/ubaniak/qail/internal/config"

func DisplayConfig(cfg config.Config) {
	headers := []string{"Name", "Value"}
	var rows [][]string
	rows = append(rows, []string{"root", cfg.Root})
	rows = append(rows, []string{"default editor", cfg.DefaultEditor})
	displayTable(headers, rows)
	if len(cfg.Editors) > 0 {
		DisplayEditors(cfg.Editors, cfg.DefaultEditor)
	}
}

// DisplayEditors prints registered editors and marks the default with "*".
func DisplayEditors(editors []config.Editor, def string) {
	headers := []string{"Name", "Command", "Default"}
	var rows [][]string
	for _, e := range editors {
		marker := ""
		if e.Name == def {
			marker = "*"
		}
		rows = append(rows, []string{e.Name, e.Command, marker})
	}
	displayTable(headers, rows)
}
