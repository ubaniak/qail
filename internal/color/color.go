package color

import "fmt"

var (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	// blue    = "\033[34m"
	// magenta = "\033[35m"
	// cyan    = "\033[36m"
	// gray    = "\033[37m"
	// white   = "\033[97m"
)

func color(c, s string) string {
	return fmt.Sprintf("%s%s%s", c, s, reset)
}

func Red(s string) string {
	return color(red, s)
}

func Green(s string) string {
	return color(green, s)
}

func Yellow(s string) string {
	return color(yellow, s)
}
