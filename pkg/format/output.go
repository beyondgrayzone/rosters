package format

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/fatih/color"
)

var (
	Brand  = color.New(color.FgHiGreen)
	Accent = color.New(color.FgHiYellow)
	Muted  = color.New(color.FgHiBlack)
	Error  = color.New(color.FgHiRed)

	quietMode  = false
	jsonMode   = false
	formatMode = "markdown"
	ansiRegex  = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func AccentBold(s string) string {
	return Accent.Add(color.Bold).Sprint(s)
}

func SetQuiet(v bool) {
	quietMode = v
}

func SetJSONMode(v bool) {
	jsonMode = v
}

func SetFormat(mode string) {
	formatMode = mode
	if mode == "plain" {
		color.NoColor = true
	}
	if mode == "json" {
		jsonMode = true
	}
}

func GetFormat() string {
	return formatMode
}

func OutputJSON(data any) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}

func PrintSuccess(msg string) {
	if quietMode || jsonMode {
		return
	}
	Brand.Printf("✓ %s\n", msg)
}

func PrintError(msg string) {
	color.New(color.FgRed).Printf("✗ %s\n", msg)
}

func PrintWarning(msg string) {
	if jsonMode {
		return
	}
	color.New(color.FgYellow).Printf("! %s\n", msg)
}

func PrintTiming(d time.Duration) {
	if !quietMode {
		Muted.Fprintf(os.Stderr, "Done in %dms\n", d.Milliseconds())
	}
}
