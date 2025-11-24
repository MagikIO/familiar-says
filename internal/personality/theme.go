package personality

import (
	"github.com/charmbracelet/lipgloss"
)

// Mood represents different emotional states
type Mood string

const (
	MoodHappy     Mood = "happy"
	MoodSad       Mood = "sad"
	MoodAngry     Mood = "angry"
	MoodSurprised Mood = "surprised"
	MoodBored     Mood = "bored"
	MoodExcited   Mood = "excited"
	MoodNeutral   Mood = "neutral"
	MoodSleepy    Mood = "sleepy"
)

// Expression defines the eyes and tongue for a mood
type Expression struct {
	Eyes   string
	Tongue string
}

// Theme represents a personality theme with colors and styles
type Theme struct {
	Name           string
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	AccentColor    lipgloss.Color
	BubbleStyle    lipgloss.Style
	CharacterStyle lipgloss.Style
	Expressions    map[Mood]Expression
}

// GetExpression returns the expression for a given mood
func (t *Theme) GetExpression(mood Mood) Expression {
	if expr, ok := t.Expressions[mood]; ok {
		return expr
	}
	return t.Expressions[MoodNeutral]
}

// Predefined themes
var (
	ThemeDefault = Theme{
		Name:           "default",
		PrimaryColor:   lipgloss.Color("15"),
		SecondaryColor: lipgloss.Color("8"),
		AccentColor:    lipgloss.Color("12"),
		BubbleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		CharacterStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		Expressions: map[Mood]Expression{
			MoodHappy:     {Eyes: "^^", Tongue: "  "},
			MoodSad:       {Eyes: "TT", Tongue: "  "},
			MoodAngry:     {Eyes: "><", Tongue: "  "},
			MoodSurprised: {Eyes: "OO", Tongue: "  "},
			MoodBored:     {Eyes: "--", Tongue: "  "},
			MoodExcited:   {Eyes: "**", Tongue: "  "},
			MoodNeutral:   {Eyes: "oo", Tongue: "  "},
			MoodSleepy:    {Eyes: "..", Tongue: "  "},
		},
	}

	ThemeRainbow = Theme{
		Name:           "rainbow",
		PrimaryColor:   lipgloss.Color("196"),
		SecondaryColor: lipgloss.Color("226"),
		AccentColor:    lipgloss.Color("51"),
		BubbleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
		CharacterStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("51")),
		Expressions: map[Mood]Expression{
			MoodHappy:     {Eyes: "◕‿◕", Tongue: "  "},
			MoodSad:       {Eyes: "•́︵•̀", Tongue: "  "},
			MoodAngry:     {Eyes: "ಠ_ಠ", Tongue: "  "},
			MoodSurprised: {Eyes: "◉_◉", Tongue: "  "},
			MoodBored:     {Eyes: "-_-", Tongue: "  "},
			MoodExcited:   {Eyes: "✧✧", Tongue: "  "},
			MoodNeutral:   {Eyes: "○○", Tongue: "  "},
			MoodSleepy:    {Eyes: "-..-", Tongue: "  "},
		},
	}

	ThemeCyber = Theme{
		Name:           "cyber",
		PrimaryColor:   lipgloss.Color("46"),
		SecondaryColor: lipgloss.Color("40"),
		AccentColor:    lipgloss.Color("82"),
		BubbleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		CharacterStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true),
		Expressions: map[Mood]Expression{
			MoodHappy:     {Eyes: "[^_^]", Tongue: "  "},
			MoodSad:       {Eyes: "[;_;]", Tongue: "  "},
			MoodAngry:     {Eyes: "[>_<]", Tongue: "  "},
			MoodSurprised: {Eyes: "[O_O]", Tongue: "  "},
			MoodBored:     {Eyes: "[-_-]", Tongue: "  "},
			MoodExcited:   {Eyes: "[*_*]", Tongue: "  "},
			MoodNeutral:   {Eyes: "[o_o]", Tongue: "  "},
			MoodSleepy:    {Eyes: "[._.]", Tongue: "  "},
		},
	}

	ThemeRetro = Theme{
		Name:           "retro",
		PrimaryColor:   lipgloss.Color("214"),
		SecondaryColor: lipgloss.Color("208"),
		AccentColor:    lipgloss.Color("202"),
		BubbleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		CharacterStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		Expressions: map[Mood]Expression{
			MoodHappy:     {Eyes: ":D", Tongue: "  "},
			MoodSad:       {Eyes: ":(", Tongue: "  "},
			MoodAngry:     {Eyes: ">:(", Tongue: "  "},
			MoodSurprised: {Eyes: ":O", Tongue: "  "},
			MoodBored:     {Eyes: ":|", Tongue: "  "},
			MoodExcited:   {Eyes: ":)", Tongue: "  "},
			MoodNeutral:   {Eyes: ":-", Tongue: "  "},
			MoodSleepy:    {Eyes: "z_z", Tongue: "  "},
		},
	}
)

// GetTheme returns a theme by name
func GetTheme(name string) Theme {
	switch name {
	case "rainbow":
		return ThemeRainbow
	case "cyber":
		return ThemeCyber
	case "retro":
		return ThemeRetro
	default:
		return ThemeDefault
	}
}

// AllThemes returns a list of all available themes
func AllThemes() []string {
	return []string{"default", "rainbow", "cyber", "retro"}
}

// AllMoods returns a list of all available moods
func AllMoods() []Mood {
	return []Mood{
		MoodNeutral,
		MoodHappy,
		MoodSad,
		MoodAngry,
		MoodSurprised,
		MoodBored,
		MoodExcited,
		MoodSleepy,
	}
}
