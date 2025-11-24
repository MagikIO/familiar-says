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

- Go 1.19 or higher
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
│   ├── animation/         # Terminal animations
│   ├── bubble/            # Speech bubble generation
│   ├── character/         # Character rendering
│   ├── effects/           # Visual effects
│   └── personality/       # Themes and moods
├── pkg/
│   └── cowparser/         # .cow file parser
├── characters/            # Character files (.cow)
├── examples/              # Example scripts
└── main.go               # Entry point
```

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
