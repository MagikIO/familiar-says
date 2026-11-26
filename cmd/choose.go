package cmd

import (
	"fmt"
	"os"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/character"
	"github.com/MagikIO/familiar-says/internal/config"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/live"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var choosePrint bool

var chooseCmd = &cobra.Command{
	Use:     "choose [character]",
	Aliases: []string{"pick"},
	Short:   "Meet every familiar and choose yours",
	Long: `Opens a gallery of every familiar, each one alive on a small stage, and saves
the one you pick as your familiar in ~/.config/familiar-says/config.json. It
is then used by plain "familiar-says", by "familiar-says live", and by the
prompt perch. Familiars that are already running change shape on the spot.

Name a character to choose it without the gallery:

    familiar-says choose owl

With --print the choice is written to stdout instead of the config file, for
use in scripts:

    familiar-says "hi" --character (familiar-says choose --print)`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config file error: %w", err)
		}
		if cfg == nil {
			cfg = &config.Config{}
		}
		current, themeName := deref(cfg.Default.Character, ""), deref(cfg.Default.Theme, "")
		if cfg.Live != nil {
			current = firstSet(deref(cfg.Live.Character, ""), current)
			themeName = firstSet(deref(cfg.Live.Theme, ""), themeName)
		}

		var name string
		if len(args) == 1 {
			char, err := character.LoadCharacter(args[0])
			if err != nil {
				return fmt.Errorf("%w (see them all with `familiar-says choose`)", err)
			}
			name = char.Name
		} else {
			// The gallery draws on stderr's terminal when stdout is being
			// captured, which is what makes `(familiar-says choose --print)` work.
			if !term.IsTerminal(int(os.Stdin.Fd())) || (!choosePrint && !term.IsTerminal(int(os.Stdout.Fd()))) {
				return fmt.Errorf("the gallery needs a terminal; name one instead: familiar-says choose <character> (list them with --list-characters)")
			}
			var chars []*canvas.Character
			for _, n := range character.ListCharacters() {
				if c, err := character.LoadCharacter(n); err == nil {
					chars = append(chars, c)
				}
			}
			name, err = live.RunGallery(chars, personality.GetTheme(themeName), current, choosePrint)
			if err != nil {
				return err
			}
			if name == "" {
				return nil // backed out; nothing changes
			}
		}

		if choosePrint {
			fmt.Println(name)
			return nil
		}
		path, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}
		if err := config.SetCharacter(path, name); err != nil {
			return err
		}
		fmt.Printf("Your familiar is now the %s. (saved to %s)\n", name, path)
		// Familiars that are already running change shape on the spot.
		if n := ipc.Broadcast(ipc.Event{Type: ipc.TypeBecome, Text: name}); n > 0 {
			fmt.Printf("%s changed shape.\n", plural(n, "running familiar"))
		}
		return nil
	},
}

func init() {
	chooseCmd.Flags().BoolVar(&choosePrint, "print", false, "Print the chosen name instead of saving it")
	rootCmd.AddCommand(chooseCmd)
}
