package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/shell"
	"github.com/spf13/cobra"
)

var (
	sendSocket string

	evExit     int
	evDuration string
	evCwd      string
	evShell    string
	evCols     string
	evSeq      int64
	evOut      string

	tellMood  string
	tellStyle string

	hookPerch     bool
	hookCharacter string
	hookTheme     string
	hookQuiet     bool
	hookSignal    string
)

func socketPath() string {
	if sendSocket != "" {
		return sendSocket
	}
	return ipc.SocketPath()
}

// send delivers an event to the running familiar on behalf of a command the
// user typed, so "nobody is listening" is reported rather than swallowed.
func send(ev ipc.Event) error {
	return ipc.Send(socketPath(), ev)
}

// atoi reads a number that a shell variable may have left empty.
func atoi(s string) int64 {
	var n int64
	fmt.Sscan(strings.TrimSpace(s), &n)
	return n
}

var eventCmd = &cobra.Command{
	Use:    "event <type> [-- command line]",
	Short:  "Report a shell event to the running familiar (used by the shell hooks)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	// Hooks run this on every command, in every shell, whether or not a
	// familiar is up. It must never print or fail in a way the user sees.
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		if evCwd == "" {
			evCwd, _ = os.Getwd()
		}
		err := send(ipc.Event{
			Type:       args[0],
			Cmd:        strings.Join(args[1:], " "),
			Exit:       evExit,
			DurationMs: atoi(evDuration),
			Cwd:        evCwd,
			Shell:      evShell,
			Cols:       int(atoi(evCols)),
			Seq:        evSeq,
			Transcript: evOut,
		})
		if errors.Is(err, ipc.ErrNotRunning) {
			return nil
		}
		return err
	},
}

var tellCmd = &cobra.Command{
	Use:   "tell <message>",
	Short: "Have your running familiar say something",
	Example: `  familiar-says tell "Deploy finished"
  make build && familiar-says tell --mood excited "It built!"`,
	Args:          cobra.MinimumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		return send(ipc.Event{Type: ipc.TypeSay, Text: strings.Join(args, " "), Mood: tellMood, Style: tellStyle})
	},
}

var askCmd = &cobra.Command{
	Use:           "ask <question>",
	Short:         "Ask your running familiar a question (needs live.oracle in the config)",
	Args:          cobra.MinimumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		// The question is about wherever it is asked from.
		cwd, _ := os.Getwd()
		return send(ipc.Event{Type: ipc.TypeAsk, Text: strings.Join(args, " "), Cwd: cwd})
	},
}

func simpleEvent(use, short, typ string) *cobra.Command {
	return &cobra.Command{
		Use: use, Short: short, Args: cobra.NoArgs,
		SilenceErrors: true, SilenceUsage: true,
		RunE: func(*cobra.Command, []string) error { return send(ipc.Event{Type: typ}) },
	}
}

var hookCmd = &cobra.Command{
	Use:   "hook <fish|bash|zsh>",
	Short: "Print shell integration so your familiar can see what you run",
	Long: `Prints a script that reports each command you run (its text, exit status,
duration and directory) to your running familiar, and binds Alt-G to accept
the familiar's latest suggestion. The reports go to a local Unix socket and
nowhere else.

    familiar-says hook fish | source        # config.fish
    eval "$(familiar-says hook bash)"       # ~/.bashrc
    eval "$(familiar-says hook zsh)"        # ~/.zshrc

With --perch (fish 4.1+ or zsh), the familiar also lives on your prompt: it
sits above the line you are typing on, animated, and folds away when you run
a command. No split or second window needed.

    familiar-says hook fish --perch | source
    eval "$(familiar-says hook zsh --perch)"`,
	Args:          cobra.ExactArgs(1),
	ValidArgs:     shell.Supported,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		bin, err := os.Executable()
		if err != nil {
			bin = "familiar-says"
		}
		opt := shell.Options{Bin: bin, Dir: ipc.RuntimeDir(), Perch: hookPerch, Signal: hookSignal}
		if hookCharacter != "" {
			opt.PerchArgs = append(opt.PerchArgs, "--character", hookCharacter)
		}
		if hookTheme != "" {
			opt.PerchArgs = append(opt.PerchArgs, "--theme", hookTheme)
		}
		if hookQuiet {
			opt.PerchArgs = append(opt.PerchArgs, "--quiet")
		}
		script, err := shell.Script(args[0], opt)
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	},
}

func init() {
	eventCmd.Flags().IntVar(&evExit, "exit", 0, "Exit status of the command")
	eventCmd.Flags().StringVar(&evDuration, "duration", "", "How long the command ran, in milliseconds")
	eventCmd.Flags().StringVar(&evCwd, "cwd", "", "Directory the command ran in")
	eventCmd.Flags().StringVar(&evShell, "shell", "", "Name of the reporting shell")
	eventCmd.Flags().StringVar(&evCols, "cols", "", "Width of the shell's terminal")
	eventCmd.Flags().Int64Var(&evSeq, "seq", 0, "Command sequence number")
	eventCmd.Flags().StringVar(&evOut, "transcript", "", "File the command's output was recorded to")

	tellCmd.Flags().StringVarP(&tellMood, "mood", "m", "", "Mood to say it with (happy, sad, angry, ...)")
	tellCmd.Flags().StringVar(&tellStyle, "bubble-style", "", "Bubble style (say, think, shout, whisper, song)")

	hookCmd.Flags().BoolVar(&hookPerch, "perch", false, "Put the familiar on your prompt (fish and zsh)")
	hookCmd.Flags().StringVarP(&hookCharacter, "character", "c", "", "Character for the perched familiar")
	hookCmd.Flags().StringVarP(&hookTheme, "theme", "t", "", "Theme for the perched familiar")
	hookCmd.Flags().BoolVarP(&hookQuiet, "quiet", "q", false, "Perched familiar only speaks when it is useful")
	hookCmd.Flags().StringVar(&hookSignal, "signal", "USR2", "Signal the perch uses to repaint the prompt (USR1 or USR2)")

	senders := []*cobra.Command{
		eventCmd, tellCmd, askCmd,
		simpleEvent("pet", "Pet your running familiar", ipc.TypePet),
		simpleEvent("nap", "Send your running familiar to sleep", ipc.TypeNap),
		simpleEvent("wake", "Wake your running familiar", ipc.TypeWake),
	}
	for _, c := range senders {
		c.Flags().StringVar(&sendSocket, "socket", "", "Socket of the familiar to talk to")
		rootCmd.AddCommand(c)
	}
	rootCmd.AddCommand(hookCmd)
}
