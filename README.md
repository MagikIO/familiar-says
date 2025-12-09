# familiar-says

[![Test Suite](https://github.com/MagikIO/familiar-says/actions/workflows/test.yml/badge.svg)](https://github.com/MagikIO/familiar-says/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MagikIO/familiar-says)](https://goreportcard.com/report/github.com/MagikIO/familiar-says)

A modern, feature-rich replacement for `cowsay` built in Go. Create personality-themed speech bubbles with rich styling, animations, and visual effects.

## Features

- 🎨 **Rich Styling** - Beautiful terminal output using [Charm Lipgloss](https://github.com/charmbracelet/lipgloss)
- 😊 **Mood-Based Expressions** - Multiple emotional states (happy, sad, angry, surprised, etc.)
- ⌨️ **Typing Animations** - Watch your messages appear character by character
- 🎭 **Personality Themes** - Default, Rainbow, Cyber, and Retro themes with unique color schemes
- 🎆 **Visual Effects** - Confetti, fireworks, sparkles, and rainbow effects
- 🐮 **Custom Characters** - Support for traditional .cow character files
- 💭 **Bubble Styles** - Both speech and thought bubbles
- 🎪 **Multi-Panel Layouts** - Display multiple characters at once (experimental)

## Installation

### From Source

```bash
git clone https://github.com/MagikIO/familiar-says.git
cd familiar-says
go build -o familiar-says .
```

### Using Go Install

```bash
go install github.com/MagikIO/familiar-says@latest
```

## Quick Start

Basic usage:

```bash
familiar-says "Hello, World!"
```

With mood and theme:

```bash
familiar-says --mood happy --theme rainbow "I'm so happy!"
```

With animation:

```bash
familiar-says --animate "Watch me type..."
```

With effects:

```bash
familiar-says --effect confetti "Celebration time!"
```

Using a custom character:

```bash
familiar-says --character characters/cat.cow "Meow!"
```

## Usage

```
familiar-says [message] [flags]

Flags:
  -a, --animate           Enable typing animation
  -c, --character string  Path to .cow character file
  -e, --effect string     Visual effect (none, confetti, fireworks, sparkle, rainbow) (default "none")
  -E, --list-effects      List available effects
  -M, --list-moods        List available moods
  -T, --list-themes       List available themes
  -m, --mood string       Mood expression (happy, sad, angry, surprised, bored, excited, neutral, sleepy) (default "neutral")
  -p, --multipanel        Enable multi-panel mode (experimental)
  -s, --speed int         Animation speed in milliseconds (default 50)
  -t, --theme string      Theme to use (default, rainbow, cyber, retro) (default "default")
      --think             Use thought bubble instead of speech bubble
  -w, --width int         Width of speech bubble (default 40)
  -h, --help              help for familiar-says
```

## Themes

### Default
Classic terminal colors with standard ASCII expressions
- Eyes: `oo`, `^^`, `TT`, etc.

### Rainbow
Colorful Unicode expressions with vibrant colors
- Eyes: `◕‿◕`, `•́︵•̀`, `ಠ_ಠ`, etc.

### Cyber
Green matrix-style with boxed expressions
- Eyes: `[^_^]`, `[;_;]`, `[>_<]`, etc.

### Retro
Orange/amber colors with classic emoticons
- Eyes: `:D`, `:(`, `>:(`, etc.

## Moods

- **neutral** - Default calm expression
- **happy** - Joyful expression
- **sad** - Sorrowful expression
- **angry** - Upset expression
- **surprised** - Shocked expression
- **bored** - Uninterested expression
- **excited** - Enthusiastic expression
- **sleepy** - Tired expression

Each theme has unique eye and tongue expressions for each mood!

## Effects

### Confetti
Adds colorful confetti characters around the output
```bash
familiar-says --effect confetti "Party time!"
```

### Fireworks
Creates firework-like star bursts
```bash
familiar-says --effect fireworks "Boom!"
```

### Sparkle
Adds sparkle emojis around the output
```bash
familiar-says --effect sparkle "Shiny!"
```

### Rainbow
Colors each character with rainbow colors
```bash
familiar-says --effect rainbow "Colorful!"
```

## Custom Characters

familiar-says supports traditional `.cow` character files. Several example characters are included:

- `default.cow` - Classic cow
- `cat.cow` - Cute cat
- `bunny.cow` - Adorable bunny
- `dragon.cow` - Majestic dragon

Create your own .cow files following the standard format:

```perl
## My Custom Character
$the_cow = <<EOC;
        $thoughts   Your
         $thoughts  Character
            $eyes   Here!
            $tongue
EOC
```

## Examples

### Combine features:

```bash
# Happy cat with rainbow effect
familiar-says --character characters/cat.cow --mood happy --effect rainbow "I love rainbows!"

# Animated angry cyber-themed message
familiar-says --mood angry --theme cyber --animate --speed 30 "System breach detected!"

# Thought bubble with sparkles
familiar-says --think --effect sparkle "What should I think about?"

# Custom width for long messages
familiar-says --width 60 "This is a much longer message that needs more space to display properly without wrapping too much."
```

### Piping input:

```bash
echo "Hello from stdin!" | familiar-says

fortune | familiar-says --mood happy --theme rainbow
```

## Architecture

The project is organized into several packages:

- `pkg/cowparser` - Parses .cow character files
- `internal/bubble` - Speech bubble generation with word wrapping
- `internal/personality` - Theme and mood system
- `internal/animation` - Terminal animations using Bubble Tea
- `internal/effects` - Visual effects engine
- `internal/character` - Character rendering engine
- `internal/canvas` - Low-level character rendering with color support
- `internal/errors` - Custom error types with consistent formatting
- `cmd` - CLI application using Cobra

### Error Handling

The application features comprehensive error handling with:

- **Custom error types** for better error context (CharacterLoadError, ColorParseError, ValidationError)
- **Wrapped errors** using `fmt.Errorf` with `%w` for full error chains
- **Input validation** at entry points with user-friendly messages
- **Graceful degradation** for terminal width detection failures
- **Clear error messages** showing exactly what went wrong and where

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework

## Contributing

Contributions are welcome! Feel free to:
- Add new themes
- Create new character files
- Implement new effects
- Improve animations
- Fix bugs

## License

MIT License - see LICENSE file for details

## Credits

Inspired by the classic `cowsay` by Tony Monroe. Built with ❤️ using [Charm](https://charm.sh/) libraries.
