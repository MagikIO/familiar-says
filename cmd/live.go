package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/character"
	"github.com/MagikIO/familiar-says/internal/config"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/live"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Flags shared by the commands that bring a familiar to life.
var (
	liveCharacter string
	liveTheme     string
	liveSocket    string
	liveQuiet     bool

	perchPID    int
	perchCols   int
	perchSignal string
	perchFormat string
)

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "Bring your familiar to life in this terminal",
	Long: `Runs your familiar as a live companion that fills this terminal: it wanders,
fidgets, naps, and reacts to what you do in your other shells.

Give it a split or a spare window, then load the shell hook in the shells you
work in so it can see what you run:

    familiar-says hook fish | source        # config.fish
    eval "$(familiar-says hook bash)"       # ~/.bashrc
    eval "$(familiar-says hook zsh)"        # ~/.zshrc

It corrects typos, explains exit codes, announces long commands, briefs you
on projects you cd into, and Alt-G in your shell accepts its suggestion.

Keys: space pet · t talk · a ask · e explain last failure · y copy suggestion
      z nap · m mute · enter skip · q quit`,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		world, err := newWorld(cmd, false, "cat")
		if err != nil {
			return err
		}
		path := liveSocket
		if path == "" {
			path = filepath.Join(ipc.RuntimeDir(), "live.sock")
		}
		server, err := ipc.Listen(path)
		if err != nil {
			return err
		}
		defer server.Close()
		hints := live.NewHints(filepath.Join(filepath.Dir(path), "live.hint"))
		defer hints.Close()
		defer world.Brain.Memory().Save()
		return live.RunTUI(world, server, hints)
	},
}

var perchCmd = &cobra.Command{
	Use:           "perch",
	Short:         "Run the headless familiar behind a shell prompt (started by `hook fish --perch`)",
	Hidden:        true,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The perch signals its shell to repaint. A shell that has not
		// loaded the hook has no handler for that signal and would be killed
		// by it, so refuse anything that looks like a manual launch.
		if perchPID <= 0 || term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("perch is started by the shell hook: add `familiar-says hook fish --perch | source` to config.fish")
		}
		sig, err := live.ParseSignal(perchSignal)
		if err != nil {
			return err
		}
		live.UseEnvColor()
		world, err := newWorld(cmd, true, "owl")
		if err != nil {
			return err
		}
		// The perch is started by the shell, in the shell's directory: that
		// is where the user is until a command says otherwise.
		if cwd, err := os.Getwd(); err == nil {
			world.Brain.At(cwd)
		}
		base := filepath.Join(ipc.RuntimeDir(), "shell-"+strconv.Itoa(perchPID))
		server, err := ipc.Listen(base + ".sock")
		if err != nil {
			return err
		}
		defer server.Close()
		defer world.Brain.Memory().Save()
		return live.RunPerch(live.PerchOptions{
			World:     world,
			Server:    server,
			Hints:     live.NewHints(base + ".hint"),
			FramePath: base + ".frame",
			ShellPID:  perchPID,
			Signal:    sig,
			Cols:      perchCols,
			Format:    perchFormat,
		})
	},
}

func init() {
	for _, c := range []*cobra.Command{liveCmd, perchCmd} {
		c.Flags().StringVarP(&liveCharacter, "character", "c", "", "Which familiar to bring to life")
		c.Flags().StringVarP(&liveTheme, "theme", "t", "", "Theme to use (default, rainbow, cyber, retro)")
		c.Flags().BoolVarP(&liveQuiet, "quiet", "q", false, "Only speak when there is something useful to say")
	}
	liveCmd.Flags().StringVar(&liveSocket, "socket", "", "Socket to listen on (default: the shared live socket)")
	perchCmd.Flags().IntVar(&perchPID, "pid", 0, "PID of the shell to perch on")
	perchCmd.Flags().IntVar(&perchCols, "cols", 80, "Width of the shell's terminal")
	perchCmd.Flags().StringVar(&perchSignal, "signal", "USR2", "Signal that asks the shell to repaint")
	perchCmd.Flags().StringVar(&perchFormat, "format", "", "Frame format: empty for fish, \"zsh\" for zsh prompt syntax")
	rootCmd.AddCommand(liveCmd, perchCmd)
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// firstSet returns the first non-empty string.
func firstSet(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// newWorld assembles a World from flags, environment and the config file.
// compact selects the short, prompt-sized stage.
func newWorld(_ *cobra.Command, compact bool, fallbackCharacter string) (*live.World, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config file error: %w", err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	lc := cfg.Live
	if lc == nil {
		lc = &config.LiveConfig{}
	}
	env := config.LoadFromEnv()

	name := firstSet(liveCharacter, deref(env.Character, ""), deref(lc.Character, ""), deref(cfg.Default.Character, ""), fallbackCharacter)
	char, err := character.LoadCharacter(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load character: %w", err)
	}
	theme := personality.GetTheme(firstSet(liveTheme, deref(env.Theme, ""), deref(lc.Theme, ""), deref(cfg.Default.Theme, "")))

	var colors *canvas.CharacterColors
	if d := cfg.Default; d.OutlineColor != nil || d.EyeColor != nil || d.MouthColor != nil {
		colors = &canvas.CharacterColors{
			Outline: deref(d.OutlineColor, ""),
			Eyes:    deref(d.EyeColor, ""),
			Mouth:   deref(d.MouthColor, ""),
		}
	}

	bc := brain.DefaultConfig()
	bc.Talkativeness = deref(lc.Talkativeness, bc.Talkativeness)
	if liveQuiet {
		bc.Talkativeness = 0
	}
	bc.Tips = deref(lc.Tips, bc.Tips)
	if lc.LongCommandSeconds != nil {
		bc.LongCommand = time.Duration(*lc.LongCommandSeconds) * time.Second
	}
	if lc.BreakMinutes != nil {
		bc.BreakEvery = time.Duration(*lc.BreakMinutes) * time.Minute
	}
	if o := lc.Oracle; o != nil && len(o.Command) > 0 {
		bc.Oracle = brain.Oracle{Command: o.Command, Timeout: time.Duration(o.TimeoutSeconds) * time.Second}
		bc.ShareContext = deref(o.ShareContext, true)
	}

	bc.Notify = deref(lc.Notifications, bc.Notify)
	bc.GitHub = deref(lc.GitHub, false)
	if len(lc.Rules) > 0 {
		if err := json.Unmarshal(lc.Rules, &bc.Rules); err != nil {
			return nil, fmt.Errorf("live.rules in the config file: %w", err)
		}
	}

	width := deref(lc.BubbleWidth, 40)
	if compact && lc.BubbleWidth == nil {
		width = 56 // a prompt-height stage has room to spare sideways, not upward
	}
	return live.NewWorld(live.Options{
		Character:     char,
		Theme:         theme,
		Colors:        colors,
		Brain:         bc,
		Memory:        brain.LoadMemory(brain.MemoryPath()),
		BubbleWidth:   width,
		Scenery:       deref(lc.Scenery, true),
		Compact:       compact,
		Restless:      deref(lc.Restlessness, 1),
		NapAfter:      time.Duration(deref(lc.NapMinutes, 5)) * time.Minute,
		Seed:          time.Now().UnixNano() ^ int64(os.Getpid()),
		LoadCharacter: character.LoadCharacter,
	}), nil
}
