package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/character"
	"github.com/MagikIO/familiar-says/internal/config"
	"github.com/MagikIO/familiar-says/internal/live"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var newFrom string

// blank is the starting point for a character drawn from scratch.
const blank = `{
  "name": %q,
  "description": "A familiar of my own",
  "art": [
    "  .-.  ",
    " (@@ ) ",
    " /| |\\ ",
    "  ' '  "
  ],
  "anchor": {"x": 3, "y": 0},
  "eyes": {"line": 1, "col": 2, "width": 2, "placeholder": "@@"},
  "colors": {"outline": "#87AFFF", "eyes": "#FFD75F"},
  "expressions": {
    "happy": {"eyes": "^^"},
    "closed": {"eyes": "--"}
  },
  "live": {"facing": "left", "locomotion": "walk", "speed": 6, "fidgets": ["wave"], "sounds": ["Hello."]},
  "animations": {
    "wave": {
      "frames": [
        {"duration": 150, "art": ["  .-.  ", " (@@ ) ", " /| |/ ", "  ' '  "]},
        {"duration": 150, "art": ["  .-. /", " (@@ ) ", " /| |  ", "  ' '  "]},
        {"duration": 150, "art": ["  .-.  ", " (@@ ) ", " /| |/ ", "  ' '  "]},
        {"duration": 150, "art": ["  .-.  ", " (@@ ) ", " /| |\\ ", "  ' '  "]}
      ],
      "loop": false
    }
  }
}
`

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Start drawing a familiar of your own",
	Long: `Creates a character file in ~/.config/familiar-says/characters/ for you to
edit, either from a small blank creature or, with --from, as a copy of one of
the built-in familiars.

Then open it on the workbench, which shows it alive and reloads every time
you save:

    familiar-says new moth
    familiar-says workshop moth

Your characters appear in "familiar-says choose" alongside the built-in ones.`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		name := strings.ToLower(args[0])
		if !validName.MatchString(name) {
			return fmt.Errorf("%q won't do as a name: use lowercase letters, digits, - and _", args[0])
		}
		dir := character.UserDir()
		path := filepath.Join(dir, name+".json")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; open it with `familiar-says workshop %s`", path, name)
		}
		source := []byte(fmt.Sprintf(blank, name))
		if newFrom != "" {
			data, ok := canvas.BuiltinSource(newFrom)
			if !ok {
				return fmt.Errorf("there is no built-in familiar called %q to start from", newFrom)
			}
			// Rename the copy without disturbing how the file is laid out.
			var probe struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(data, &probe)
			was, _ := json.Marshal(probe.Name)
			now, _ := json.Marshal(name)
			source = []byte(strings.Replace(string(data), `"name": `+string(was), `"name": `+string(now), 1))
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, source, 0o644); err != nil {
			return err
		}
		fmt.Printf("Created %s\nOpen it on the workbench:  familiar-says workshop %s\n", path, name)
		return nil
	},
}

var workshopCmd = &cobra.Command{
	Use:   "workshop <name|file.json>",
	Short: "See a character alive while you edit it",
	Long: `Opens a character file on the workbench: the familiar lives on a small stage,
and every time you save the file it is reloaded, so you can draw in your
editor on one side and watch the result on the other. Anything wrong with
the file is listed underneath in plain words.

Keys: 1-9 play an animation · m cycle moods · r cycle reactions · f flip
      w wander · t talk · z nap · l reload now · q quit`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		path := args[0]
		if _, err := os.Stat(path); err != nil {
			mine := filepath.Join(character.UserDir(), strings.TrimSuffix(args[0], ".json")+".json")
			if _, err := os.Stat(mine); err != nil {
				if _, builtin := canvas.BuiltinSource(args[0]); builtin {
					return fmt.Errorf("%s is built in and can't be edited in place; make your own copy with `familiar-says new my-%s --from %s`", args[0], args[0], args[0])
				}
				return fmt.Errorf("no character file %q (start one with `familiar-says new %s`)", args[0], args[0])
			}
			path = mine
		}
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			// No terminal to draw on: still useful as a checker.
			char, err := canvas.LoadCharacter(path)
			if err != nil {
				return err
			}
			problems := canvas.Lint(char)
			for _, p := range problems {
				fmt.Println("! " + p)
			}
			if len(problems) > 0 {
				return fmt.Errorf("%d problem(s) in %s", len(problems), path)
			}
			fmt.Println("no problems found")
			return nil
		}
		theme := ""
		if cfg, _ := config.Load(); cfg != nil {
			theme = deref(cfg.Default.Theme, "")
		}
		return live.RunWorkshop(path, personality.GetTheme(theme))
	},
}

func init() {
	newCmd.Flags().StringVar(&newFrom, "from", "", "Start from a copy of this built-in familiar")
	rootCmd.AddCommand(newCmd, workshopCmd)
}
