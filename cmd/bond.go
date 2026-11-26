package cmd

import (
	"fmt"
	"strings"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var bondCmd = &cobra.Command{
	Use:   "bond",
	Short: "See how you and your familiar are getting on",
	Long: `Shows the bond between you and your familiar: how long you have been
together, what it has been through with you, and what it has found to wear.

The bond grows with days spent together, commands it has watched, failures it
saw you fix, and (up to a point) petting.`,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(*cobra.Command, []string) error {
		mem := brain.LoadMemory(brain.MemoryPath())
		b := mem.Bond()
		bold := lipgloss.NewStyle().Bold(true)
		faint := lipgloss.NewStyle().Faint(true)

		fmt.Printf("%s  %s\n", bold.Render(strings.ToUpper(b.Title[:1])+b.Title[1:]), faint.Render(plural(b.Days, "day")+" together"))
		if b.Next > 0 {
			const width = 24
			done := (b.Points - b.Floor) * width / (b.Next - b.Floor)
			bar := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF87AF")).Render(strings.Repeat("█", done)) +
				faint.Render(strings.Repeat("░", width-done))
			fmt.Printf("%s  %d / %d to the next stage\n", bar, b.Points-b.Floor, b.Next-b.Floor)
		}
		fmt.Printf("%s\n", faint.Render(fmt.Sprintf("watched %s · winced at %s · celebrated %s · petted %s",
			plural(mem.Commands, "command"), plural(mem.Failures, "failure"), plural(mem.Fixes, "fix"), plural(mem.Pets, "time"))))

		var have []string
		for _, a := range mem.Unlocked() {
			item := a.Name + " " + a.Glyph
			if a.Name == mem.Wearing {
				item = bold.Render(item + " (on)")
			}
			have = append(have, item)
		}
		switch {
		case len(have) == 0:
			fmt.Println("Nothing to wear yet. It finds its first accessory when you become companions.")
		default:
			fmt.Printf("Accessories: %s   %s\n", strings.Join(have, ", "), faint.Render("familiar-says wear <name|none>"))
		}
		return nil
	},
}

var wearCmd = &cobra.Command{
	Use:   "wear <accessory|none>",
	Short: "Have your familiar put on something it has found",
	Long: `Puts an accessory on your familiar, or takes it off with "none".
Accessories are found as your bond grows; see "familiar-says bond".`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(_ *cobra.Command, args []string) error {
		mem := brain.LoadMemory(brain.MemoryPath())
		if err := mem.Wear(args[0]); err != nil {
			return err
		}
		if err := mem.Save(); err != nil {
			return err
		}
		// Familiars that are already running change on the spot.
		ipc.Broadcast(ipc.Event{Type: ipc.TypeWear, Text: args[0]})
		if mem.Wearing == "" {
			fmt.Println("Taken off.")
		} else {
			fmt.Printf("Your familiar is wearing the %s.\n", mem.Wearing)
		}
		return nil
	},
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	if strings.HasSuffix(word, "x") {
		return fmt.Sprintf("%d %ses", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func init() {
	rootCmd.AddCommand(bondCmd, wearCmd)
}
