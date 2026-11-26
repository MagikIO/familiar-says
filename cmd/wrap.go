package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/wrap"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var wrapCmd = &cobra.Command{
	Use:   "wrap [-- command [args...]]",
	Short: "Run your shell where the familiar can read command output",
	Long: `Runs your shell (or the given command) inside a pass-through terminal that
lets the familiar read what your commands print. With that it can tell you
what an error means and how to fix it, rather than guessing from the exit
status, and it knows whether the window is focused, so it can send a desktop
notification when a long command finishes while you are elsewhere.

Everything you type and everything your programs print passes straight
through. Only the output of the most recent command is kept, in a file in
$XDG_RUNTIME_DIR that only you can read, and it is replaced by the next
command. Nothing is sent anywhere.

Try it in one terminal first:

    familiar-says wrap

To use it in every fish shell, put this at the very top of config.fish:

    if status is-interactive; and not set -q FAMILIAR_SAYS_WRAPPED
        exec familiar-says wrap -- (status fish-path)
    end

(Naming the shell matters: without it, wrap starts $SHELL, which is not
always the shell you are in.) To open one shell without it:

    FAMILIAR_SAYS_WRAPPED=0 fish`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		argv := args
		if len(argv) == 0 {
			shell := os.Getenv("SHELL")
			if shell == "" {
				shell = "/bin/sh"
			}
			argv = []string{shell}
		}
		// A familiar that cannot listen in is no reason to leave anyone
		// without a shell: if the wrapper fails, run the shell plainly.
		plain := func(why any) error {
			fmt.Fprintf(os.Stderr, "familiar-says: running %s without wrap: %v\n", argv[0], why)
			if path, lookErr := exec.LookPath(argv[0]); lookErr == nil && term.IsTerminal(int(os.Stdin.Fd())) {
				env := append(os.Environ(), wrap.EnvWrapped+"=0")
				return syscall.Exec(path, argv, env)
			}
			return fmt.Errorf("%v", why)
		}
		started := false
		defer func() {
			if r := recover(); r != nil && !started {
				_ = plain(r)
			}
		}()
		code, err := wrap.Run(argv, ipc.RuntimeDir())
		if err != nil {
			return plain(err)
		}
		started = true
		os.Exit(code)
		return nil
	},
}

var (
	trustRevoke bool
	trustYes    bool
)

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Let your familiar follow this project's " + brain.ProjectFile,
	Long: `A project can carry instructions for your familiar in a ` + brain.ProjectFile + ` file:
things to say or suggest when particular commands run here. Because that file
can arrive with a repository you cloned, the familiar does not follow it until
you have looked at it and said so.

This shows the project's rules and, if you agree, trusts that exact version of
the file. If the file changes, you will be asked again.`,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(*cobra.Command, []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		project, perr := brain.FindProject(dir)
		if project == nil {
			return fmt.Errorf("no %s here or in any directory above", brain.ProjectFile)
		}
		mem := brain.LoadMemory(brain.MemoryPath())
		if trustRevoke {
			delete(mem.Trusted, project.Path)
			if err := mem.Save(); err != nil {
				return err
			}
			ipc.Broadcast(ipc.Event{Type: ipc.TypeRescan})
			fmt.Printf("No longer following %s.\n", project.Path)
			return nil
		}
		if perr != nil {
			return perr
		}
		rules, problems := brain.CompileRules(project.Rules)

		fmt.Printf("%s\n\n", project.Path)
		if project.Welcome != "" {
			fmt.Printf("  on arrival, say: %s\n", project.Welcome)
		}
		for _, r := range rules {
			when := map[string]string{"": "runs", "always": "runs", "success": "succeeds", "failure": "fails", "before": "is about to run"}[r.On]
			fmt.Printf("  when a command matching /%s/ %s:\n", r.Match, when)
			if r.Output != "" {
				fmt.Printf("      and its output matches /%s/\n", r.Output)
			}
			if r.Say != "" {
				fmt.Printf("      say:     %s\n", r.Say)
			}
			if r.Suggest != "" {
				fmt.Printf("      suggest: %s   (put on your prompt only if you press Alt-G)\n", r.Suggest)
			}
		}
		for _, p := range problems {
			fmt.Printf("  ! %v\n", p)
		}
		fmt.Println()

		if mem.Trusted[project.Path] == project.Hash {
			fmt.Println("Already trusted.")
			return nil
		}
		if !trustYes {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("not trusting without confirmation; rerun with --yes")
			}
			fmt.Print("Let your familiar follow these? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" && answer != "yes" {
				fmt.Println("Left untrusted.")
				return nil
			}
		}
		mem.Trusted[project.Path] = project.Hash
		if err := mem.Save(); err != nil {
			return err
		}
		ipc.Broadcast(ipc.Event{Type: ipc.TypeRescan})
		fmt.Println("Trusted. Your familiar will follow them from now on.")
		return nil
	},
}

func init() {
	trustCmd.Flags().BoolVar(&trustRevoke, "revoke", false, "Stop following this project's rules")
	trustCmd.Flags().BoolVarP(&trustYes, "yes", "y", false, "Trust without asking")
	rootCmd.AddCommand(wrapCmd, trustCmd)
}
