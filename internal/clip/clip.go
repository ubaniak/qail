package clip

import (
	"fmt"

	"github.com/atotto/clipboard"

	"qail/internal/color"
)

func Cd(path string) {
	cmd := fmt.Sprintf("cd %s", path)
	fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(cmd))
	clipboard.WriteAll(cmd)
}

func Cmd(cmd string) {
	fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(cmd))
	clipboard.WriteAll(cmd)
}
