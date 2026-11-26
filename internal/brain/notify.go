package brain

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Notify raises a desktop notification, using whatever the platform offers.
// It is best effort: with no notifier installed it quietly does nothing.
func Notify(title, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "darwin":
		script := `display notification "` + osaEscape(body) + `" with title "` + osaEscape(title) + `"`
		_ = exec.CommandContext(ctx, "osascript", "-e", script).Run()
	default:
		if path, err := exec.LookPath("notify-send"); err == nil {
			_ = exec.CommandContext(ctx, path, "--app-name=familiar-says", "--", title, body).Run()
		}
	}
}

func osaEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
