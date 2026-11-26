# Contributing to familiar-says

Thank you for your interest in contributing to familiar-says! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/familiar-says.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Build and test: `make build && ./familiar-says "Test message"`
7. Commit your changes: `git commit -am 'Add some feature'`
8. Push to the branch: `git push origin feature/your-feature-name`
9. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.25 or higher
- Make (optional, but recommended)

### Building

```bash
make build
# or
go build -o familiar-says .
```

### Running Tests

```bash
make test
# or
go test ./...
```

### Running with Coverage

```bash
make test-coverage
```

## Project Structure

```
familiar-says/
├── cmd/                    # CLI command implementation
├── internal/
│   ├── animation/         # One-shot terminal animations
│   ├── bubble/            # Speech bubble templates
│   ├── canvas/            # Static composition, character loading, colors
│   ├── character/         # Character rendering
│   ├── config/            # Config file, profiles, environment
│   ├── effects/           # Visual effects
│   ├── personality/       # Themes and moods
│   ├── stage/             # Live: real-time grid, particles, typed bubbles
│   ├── creature/          # Live: the familiar's body and behaviour
│   ├── brain/             # Live: rules that turn shell events into remarks
│   ├── ipc/               # Live: Unix-socket event protocol
│   ├── live/              # Live: the world, the TUI, the prompt perch
│   ├── shell/             # Live: generated fish/bash/zsh hooks
│   └── wrap/              # Live: pass-through pty that captures command output
├── characters/            # Character files (JSON)
├── examples/              # Example scripts
└── main.go               # Entry point
```

### Working on the live familiar

- **Teaching it something new**: rules live in `internal/brain/brain.go` (`onCommand`, `onFailure`, `OnTick`), the knowledge they draw on in `suggest.go`, and what it says in `quips.go`. A rule returns a `Reaction`; set `Must` if it carries real information (it then bypasses rate limiting and the talkativeness setting) and `Suggest` if it proposes a command the user can accept with Alt-G. Add a test in `brain_test.go` — the brain takes an injected clock and random source, so rules are easy to test.
- **Teaching it a new error**: add a line to `remedies` in `internal/brain/diagnose.go` (a pattern, what to say, and optionally the fix) and a row to `TestDiagnoseKnownErrors`. More specific errors go above the general ones they resemble.
- **Giving a character a life**: add a `live` block to its JSON (see the README). Without one it is a front-facing walker that fidgets with whatever animations it has.
- **Seeing what you did**: `FAMILIAR_DUMP=1 go test ./internal/live -run Bounds -v` prints frames of every character on a small stage. For the real thing, `go build -o familiar-says . && ./familiar-says live`, then from another terminal `./familiar-says event command --exit 127 -- gti status`.
- **Changing the shell hooks**: the templates are in `internal/shell/`. They run inside people's interactive shells on every command, so keep them cheap, never let them print, and check `go test ./internal/shell` (it syntax-checks the output with each shell that is installed).

## Adding New Features

### Adding a New Theme

1. Edit `internal/personality/theme.go`
2. Create a new `Theme` struct with:
   - Name
   - Color scheme (PrimaryColor, SecondaryColor, AccentColor)
   - Styles for bubble and character
   - Expressions for all moods
3. Add the theme to the `GetTheme()` function
4. Add the theme name to `AllThemes()`
5. Add tests in `internal/personality/theme_test.go`

Example:
```go
ThemeNeon = Theme{
    Name:           "neon",
    PrimaryColor:   lipgloss.Color("199"),
    SecondaryColor: lipgloss.Color("213"),
    AccentColor:    lipgloss.Color("201"),
    BubbleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("199")),
    CharacterStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
    Expressions: map[Mood]Expression{
        MoodHappy:   {Eyes: "✧✧", Tongue: "  "},
        // ... add all moods
    },
}
```

### Adding a New Visual Effect

1. Edit `internal/effects/effects.go`
2. Add a new `Effect` constant
3. Implement the effect function (e.g., `applyMyEffect()`)
4. Add the effect to the `Apply()` switch statement
5. Add description to `GetEffectDescription()`
6. Add the effect to `AllEffects()`
7. Add tests in `internal/effects/effects_test.go`

### Adding a New Character

1. Create a new `.cow` file in the `characters/` directory
2. Follow the standard .cow format:

```perl
## Character name and description
$the_cow = <<EOC;
        $thoughts   Your
         $thoughts  Character
            $eyes   Art
            $tongue Here!
EOC
```

3. Use these variables in your character:
   - `$thoughts` - connector from bubble to character (\ or o)
   - `$eyes` - character's eyes (changes based on mood)
   - `$tongue` - character's tongue (optional)

4. Test your character:
```bash
./familiar-says --character characters/yourchar.cow "Test message"
```

### Adding New Moods

1. Edit `internal/personality/theme.go`
2. Add a new `Mood` constant
3. Add expressions for the new mood to all existing themes
4. Add the mood to `AllMoods()`
5. Add tests in `internal/personality/theme_test.go`

## Code Style

- Follow standard Go conventions
- Use `gofmt` to format your code
- Write clear, descriptive commit messages
- Add comments for exported functions and types
- Keep functions focused and small
- Write tests for new functionality

## Testing

- Write unit tests for new functions
- Ensure all tests pass before submitting PR
- Aim for high test coverage on core functionality
- Test edge cases and error conditions

## Documentation

- Update README.md if adding new features
- Add examples for new functionality
- Document command-line flags
- Update help text if needed

## Pull Request Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Include screenshots for visual changes
- Ensure all tests pass
- Keep PRs focused on a single feature or fix
- Update documentation as needed

## Ideas for Contributions

Here are some areas where contributions are welcome:

### New Features
- More personality themes
- Additional visual effects
- More character files
- Animation improvements
- Interactive mode
- Configuration file support
- Plugin system

### Improvements
- Performance optimizations
- Better error messages
- More comprehensive tests
- Documentation improvements
- CLI improvements

### Bug Fixes
- Report bugs via GitHub issues
- Include steps to reproduce
- Provide expected vs actual behavior

## Questions?

Feel free to open an issue for:
- Questions about contributing
- Feature discussions
- Bug reports
- General feedback

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Help others learn and grow

Thank you for contributing to familiar-says! 🎉
