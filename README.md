# familiar-says

[![Test Suite](https://github.com/MagikIO/familiar-says/actions/workflows/test.yml/badge.svg)](https://github.com/MagikIO/familiar-says/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MagikIO/familiar-says)](https://goreportcard.com/report/github.com/MagikIO/familiar-says)

A modern, feature-rich replacement for `cowsay` built in Go. Create personality-themed speech bubbles with rich styling, animations, and visual effects — or let your familiar move in: it can live in your terminal, watch what you run, and help.

## Features

- 🐾 **Live Familiar** - A moving, animated companion that wanders, naps, reacts to your commands, corrects typos, explains failures and offers tips ([details](#live-familiar))
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

Using a built-in character:

```bash
familiar-says --character cat "Meow!"
```

## Live Familiar

`familiar-says "hello"` prints a picture and exits. A live familiar stays. It walks around, blinks, fidgets, falls asleep when nothing is happening, and wakes up when you run something. Because your shell tells it what you ran and how it went, it can be useful as well as cute:

| When you…                                   | Your familiar…                                                                 |
| ------------------------------------------- | ------------------------------------------------------------------------------ |
| mistype a command (`gti status`)            | says ``Did you mean `git status`?`` — press **Alt-G** to put the fix on your prompt |
| mistype a subcommand or a path (`cd intenral`) | offers the nearest real one                                                 |
| get a cryptic exit code (126, 137, 139, curl 6, ssh 255…) | tells you what it means and what to try                          |
| wait on a long build                        | announces when it finishes and how long it took                                |
| finally fix the thing that failed four times | throws confetti                                                               |
| `cd` into a project                         | briefs you: project type, handy commands, branch, uncommitted and unpushed work |
| type the same long command for the sixth time | suggests an abbreviation (`abbr` in fish, `alias` elsewhere)                 |
| run `rm -rf` or `git push --force`          | sweats, and mentions `--force-with-lease`                                      |
| work for 50 minutes straight                | suggests a stretch                                                             |
| do nothing                                  | shares a shell tip, hums, or dozes off                                         |
| hit an error it recognises ([with `wrap`](#letting-it-read-the-error)) | says what the error means and offers the fix             |
| leave a long command running and look away  | sends a desktop notification when it finishes                                  |
| push a branch ([with GitHub on](#github-and-ci)) | tells you when CI goes green or red, and when reviews are waiting         |

### Choosing your familiar

```bash
familiar-says choose          # meet them all, pick one
familiar-says choose owl      # or just say who
```

The gallery shows every familiar alive on a small stage: browse with the arrow keys, press enter, and your pick is saved to the config file and used everywhere (plain `familiar-says`, `live`, and the prompt perch). Familiars that are already running change shape on the spot. `--print` prints the name instead of saving it.

There are two ways to keep it around.

### On your prompt (fish and zsh)

The familiar perches above the line you are typing on, animated, and folds away when you run a command so your scrollback stays clean. No split, no second window. Add to the end of your shell's config, after anything that sets up your prompt:

```fish
familiar-says hook fish --perch | source        # ~/.config/fish/config.fish
```

```zsh
eval "$(familiar-says hook zsh --perch)"        # ~/.zshrc
```

```
                    ,_,      ____________________________
                   (oo)   -< Did you mean `git status`? >
                   /)_)      ----------------------------
                    ""
~/Code/project > █
```

Options: `--character owl` (small characters suit a prompt best: `owl`, `bunny`, `fox`, `bat`), `--theme cyber`, `--quiet` (essentials only). In that shell, `familiar hide` / `familiar show` toggles it, and `familiar pet`, `familiar tell "…"`, `familiar ask "…"` talk to it.

In fish this needs 4.1 or newer. It works by wrapping `fish_prompt` (your own prompt is kept, and still sees the right `$status` and `$pipestatus`), turning on fish's transient prompt, and repainting when a small background process signals that the picture changed (SIGUSR2; `--signal USR1` if something else of yours uses that). Prompts that update themselves asynchronously, such as oh-my-posh, are supported.

In zsh the frame is placed above your `PROMPT` and redrawn with `zle reset-prompt`. bash cannot redraw its prompt while you type, so there the familiar lives in a pane instead.

### In its own pane (any shell)

Give it a split or a spare window:

```bash
familiar-says live                 # or: familiar-says live --character dragon
```

Then load the hook in the shells you work in, so it can see what you run:

```bash
familiar-says hook fish | source       # ~/.config/fish/config.fish
eval "$(familiar-says hook bash)"      # ~/.bashrc
eval "$(familiar-says hook zsh)"       # ~/.zshrc
```

| Key       | Action                                   |
| --------- | ---------------------------------------- |
| `space`   | Pet it                                   |
| `t`       | Make it say something                    |
| `a`       | Ask it a question (needs an oracle, below) |
| `e`       | Ask it to explain the last failed command |
| `y`       | Copy its latest suggestion to the clipboard |
| `z`       | Nap / wake                               |
| `m`       | Mute chatter (corrections still come through) |
| `enter`   | Skip or dismiss the current bubble       |
| `?`       | Show the keys                            |
| `q`       | Quit                                     |

### Talking to it from scripts

```bash
familiar-says tell "Deploy finished"
make test && familiar-says tell --mood excited "All green!"
familiar-says ask "how do I undo my last commit?"
familiar-says pet
```

### Letting it read the error

By default the familiar knows only what your shell knows: the command line and its exit status. Run your shell under `wrap` and it can also read what the command printed, which turns "exit 1, hm" into advice:

```
~/app > python app.py
ModuleNotFoundError: No module named 'requests'
              ________________________________________________
     ,_,     / Python can't import `requests`. Is the right   \
    (TT)   -\ virtualenv active?                              /
    /)_)     ------------------------------------------------
     ""
~/app > █                      (Alt-G puts `pip install requests` here)
```

It recognises common failures from git, Go, Node, Python, Rust, Docker and the system (missing modules, rejected pushes, ports in use, a full disk, things that need `sudo`…), names the tests that failed, and when a long build scrolls past, points at the first error.

```bash
familiar-says wrap            # try it in one terminal
```

To use it everywhere, put this at the very top of `config.fish` (or the equivalent `exec` in your `.zshrc`/`.bashrc`):

```fish
if status is-interactive; and not set -q FAMILIAR_SAYS_WRAPPED
    exec familiar-says wrap -- (status fish-path)
end
```

Name the shell as shown: without it, `wrap` starts `$SHELL`, which is not always the shell you are in. `FAMILIAR_SAYS_WRAPPED=0 fish` opens one shell without it.

`wrap` runs your shell inside a pass-through terminal. Keystrokes and output are relayed untouched; only the most recent command's output is kept, in a file under `$XDG_RUNTIME_DIR` that only you can read, replaced by the next command and deleted when the shell exits. If `wrap` cannot start, it runs your shell directly rather than leave you without one. It is also how the familiar learns whether the window has focus.

### Notifications

When a command that ran longer than `longCommandSeconds` finishes while the terminal is not the focused window, the familiar sends a desktop notification (`notify-send` on Linux, Notification Centre on macOS). Knowing about focus needs `wrap`. Set `"notifications"` to `"always"` to be notified regardless, or `"never"`.

### GitHub and CI

With `"github": true` under `live`, and the [`gh` CLI](https://cli.github.com) signed in, the familiar checks on the pull request for your current branch: it tells you when CI passes or fails (and which check), when the PR is approved or changes are requested, and when pull requests are waiting on your review. After a `git push`, and while checks are running, it looks every half minute (for up to twenty minutes); otherwise every five minutes, and only while you are active. It is off by default because it makes network requests in the background.

### Teaching it your own rules

Add rules under `live.rules` in the config file. Each one matches a command line with a regular expression:

```json
{
  "live": {
    "rules": [
      {"match": "^git pull", "on": "success", "say": "Pulled. Migrations may have changed.", "suggest": "make migrate"},
      {"match": "^terraform apply", "on": "before", "say": "Which workspace is this?", "act": "nervous"},
      {"match": "^make", "on": "failure", "output": "undefined reference", "say": "Linker trouble: check the library order."},
      {"match": "^deploy (\\w+)", "on": "success", "say": "{1} is live.", "act": "celebrate", "notify": true}
    ]
  }
}
```

| Field | Meaning |
| ----- | ------- |
| `match` | Regular expression the command line must match (required) |
| `on` | `success`, `failure`, `before` (as the command starts), or `always` (default: either outcome) |
| `exit` | Only for this exit status |
| `output` | Regular expression the command's output must match (needs `wrap`) |
| `say` | What to say. `{cmd}`, `{exit}`, `{duration}`, `{dir}` and capture groups `{1}`…`{9}` are filled in |
| `suggest` | A command to offer; Alt-G puts it on your prompt |
| `act` | `celebrate`, `cheer`, `wince`, `nervous`, `love`, `alert`, `wave`, `nod`, `sing`, `fume` |
| `mood`, `style` | Expression to wear, and bubble style (`say`, `think`, `shout`, `whisper`, `song`) |
| `notify` | Also send a desktop notification |
| `cooldownMinutes` | Stay quiet this long after firing |

Your rules outrank the familiar's own judgement.

**Per-project rules** go in a `.familiar.json` at the project root, with the same `rules` plus an optional `welcome` said on arrival:

```json
{
  "welcome": "Staging deploys need the VPN.",
  "rules": [{"match": "^make deploy", "on": "before", "say": "Did you run the smoke tests?", "suggest": "make smoke"}]
}
```

Because that file can arrive with a repository you cloned, the familiar will not follow it until you have looked at it: `familiar-says trust` shows the rules and asks. Trust is given to that exact content; if the file changes, you are asked again. `familiar-says trust --revoke` withdraws it.

### Your bond

The familiar keeps track of your time together. `familiar-says bond` shows it:

```
Friend  12 days together
██████████░░░░░░░░░░░░░░  128 / 300 to the next stage
watched 1204 commands · winced at 87 failures · celebrated 9 fixes · petted 31 times
Accessories: flower ✿, bow ⋈ (on)
```

The bond grows with days spent together, commands it has watched, failures it saw you fix, and (up to a daily limit) petting. Each new stage is celebrated, and with it the familiar finds something to wear: `familiar-says wear flower`, `familiar-says wear none`. As you grow closer it is also gladder to see you come back.

### Asking it questions

The familiar's own advice comes from rules, not a model: it works offline and sends nothing anywhere. If you want it to answer free-form questions too, point it at any program that reads a prompt on stdin and prints an answer:

```json
{
  "live": {
    "oracle": { "command": ["claude", "-p", "--model", "haiku", "--tools", "Read,Grep,Glob", "--setting-sources", "user", "--no-session-persistence"] }
  }
}
```

The oracle is started in the directory you are working in, so one that can read files (as above: Claude Code with read-only tools, loading only your own settings) can answer questions about the project in front of you.

The oracle is only ever run when you ask (`a`, `e`, or `familiar-says ask`). The question is sent along with your shell's name, working directory, last failed command and, under `wrap`, the last lines of its output; set `"shareContext": false` under `oracle` to send the question alone.

### Live configuration

All optional, under `"live"` in `~/.config/familiar-says/config.json`:

```json
{
  "live": {
    "character": "cat",
    "theme": "default",
    "talkativeness": 0.5,
    "restlessness": 1,
    "longCommandSeconds": 20,
    "breakMinutes": 50,
    "napMinutes": 5,
    "tips": true,
    "scenery": true,
    "notifications": "unfocused",
    "github": false,
    "rules": []
  }
}
```

`talkativeness` runs from 0 (only speaks when it has something useful to say) to 1 (chatterbox). `restlessness` 0 keeps it from wandering. `breakMinutes` 0 turns off break reminders.

### What the hooks share

The shell hook reports each command line, its exit status, duration and working directory to the familiar over a Unix socket in `$XDG_RUNTIME_DIR/familiar-says/` that only your user can open. Nothing leaves your machine unless you turn on GitHub awareness (which runs `gh`) or ask the oracle a question. It remembers how often you run each command (to suggest abbreviations), your bond, and which project rule files you have trusted in `~/.local/state/familiar-says/memory.json`.

### Drawing your own

```bash
familiar-says new moth                  # a small blank creature to start from
familiar-says new geeko --from suse     # or a copy of a built-in one
familiar-says workshop moth             # see it alive while you edit
```

`new` creates `~/.config/familiar-says/characters/<name>.json`. `workshop` puts that file on a bench: the familiar lives on a stage, and each time you save in your editor it is reloaded. Anything wrong with the file (a slot pointing at the wrong column, an animation frame with a missing line, a gait that does not exist) is listed underneath in plain words. Keys play each animation, cycle moods and reactions, flip it, and make it walk or talk. Your characters show up in `familiar-says choose` with the rest.

A character can have faces of its own, which outrank the theme's. That is how art with unusual eyes keeps its look:

```json
"eyes": {"line": 1, "col": 11, "width": 3, "placeholder": "oYo"},
"expressions": {
  "happy":  {"eyes": "^Y^"},
  "sleepy": {"eyes": "uYu"},
  "closed": {"eyes": "-Y-"}
}
```

Moods are `neutral`, `happy`, `sad`, `angry`, `surprised`, `bored`, `excited` and `sleepy`; `closed` is used for blinking and sleeping.

### Giving a character a life

Characters describe how they behave in an optional `live` block:

```json
"live": {
  "facing": "left",
  "locomotion": "hop",
  "speed": 9,
  "gait": "tail_wag",
  "fidgets": ["ear_wiggle"],
  "sounds": ["*nose twitch*", "Mip."]
}
```

`facing` is the way the art looks (`left`, `right`, or `front`); directional art is mirrored when the familiar turns around. `locomotion` is `walk`, `hop`, or `fly`. `gait` names the animation to loop while it moves, `fidgets` the ones to play when it is idle, and `sounds` are what it says when it has nothing else to say.

## Usage

```
familiar-says [message] [flags]

Flags:
  -a, --animate              Enable typing animation
  -c, --character string     Character to use (cat, owl, fox, bunny, penguin, dragon, robot, bat, turtle, default)
  -C, --list-characters      List available characters
  -e, --effect string        Visual effect (none, confetti, fireworks, sparkle, rainbow, rainbow-text) (default "none")
  -E, --list-effects         List available effects
  -M, --list-moods           List available moods
  -T, --list-themes          List available themes
  -m, --mood string          Mood expression (happy, sad, angry, surprised, bored, excited, neutral, sleepy) (default "neutral")
  -p, --multipanel           Enable multi-panel mode (experimental)
  -s, --speed int            Animation speed in milliseconds (default 50)
  -t, --theme string         Theme to use (default, rainbow, cyber, retro) (default "default")
      --think                Use thought bubble instead of speech bubble
  -w, --width int            Width of speech bubble (default 40)
      --outline-color string Color for character outline/body (hex, ANSI, or name)
      --eye-color string     Color for character eyes (hex, ANSI, or name)
      --mouth-color string   Color for character mouth (hex, ANSI, or name)
      --list-colors          List available named colors
  -h, --help                 help for familiar-says
```

## Configuration

familiar-says supports configuration files, profiles, and environment variables to reduce repetitive flag usage.

### Config File

Create a config file at `~/.config/familiar-says/config.json`:

```json
{
  "default": {
    "character": "cat",
    "theme": "cyber",
    "mood": "happy",
    "width": 50
  }
}
```

All flags can be configured:
- `character`, `theme`, `mood`, `width`, `animate`, `speed`, `effect`, `think`, `multipanel`
- `outlineColor`, `eyeColor`, `mouthColor`

### Profiles

Define multiple profiles for different use cases:

```json
{
  "default": {
    "character": "cat",
    "width": 50
  },
  "profiles": {
    "work": {
      "character": "owl",
      "theme": "cyber",
      "mood": "neutral"
    },
    "fun": {
      "character": "dragon",
      "theme": "rainbow",
      "effect": "sparkle",
      "animate": true
    },
    "presentation": {
      "character": "robot",
      "theme": "cyber",
      "width": 80,
      "animate": true,
      "speed": 40
    }
  }
}
```

Use profiles with the `--profile` flag:

```bash
familiar-says --profile work "Meeting at 3pm"
familiar-says --profile fun "Party time!"
```

### Environment Variables

Override config values using environment variables:

```bash
export FAMILIAR_SAYS_CHARACTER="owl"
export FAMILIAR_SAYS_THEME="cyber"
export FAMILIAR_SAYS_MOOD="neutral"

familiar-says "Using env vars!"
```

All flags have corresponding environment variables with the `FAMILIAR_SAYS_` prefix:
- `FAMILIAR_SAYS_CHARACTER`, `FAMILIAR_SAYS_THEME`, `FAMILIAR_SAYS_MOOD`
- `FAMILIAR_SAYS_WIDTH`, `FAMILIAR_SAYS_SPEED`
- `FAMILIAR_SAYS_ANIMATE`, `FAMILIAR_SAYS_THINK`, `FAMILIAR_SAYS_MULTIPANEL` (use `true`/`false`, `1`/`0`, or `yes`/`no`)
- `FAMILIAR_SAYS_EFFECT`
- `FAMILIAR_SAYS_OUTLINE_COLOR`, `FAMILIAR_SAYS_EYE_COLOR`, `FAMILIAR_SAYS_MOUTH_COLOR`
- `FAMILIAR_SAYS_PROFILE`

### Precedence Order

Configuration sources are applied in this order (highest priority last):

1. **Defaults** - Built-in defaults
2. **Config File** - Values from `~/.config/familiar-says/config.json`
3. **Environment Variables** - `FAMILIAR_SAYS_*` env vars
4. **CLI Flags** - Command-line flags (always win)

This allows you to set defaults in your config file, override them per-session with environment variables, and override everything with explicit CLI flags.

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

### Rainbow Text
Applies rainbow colors only to the message text (not the character)
```bash
familiar-says --effect rainbow-text "Colorful message!"
```

## Custom Characters

familiar-says includes several built-in character familiars:

- `default` - Classic cow
- `cat` - Cute cat
- `bunny` - Adorable bunny
- `dragon` - Majestic dragon
- `owl` - Wise owl
- `fox` - Clever fox
- `penguin` - Waddle penguin
- `robot` - Mechanical robot
- `bat` - Night bat
- `turtle` - Slow turtle

Characters are defined using JSON files with support for customizable colors and anchor positions. Example format:

```json
{
  "name": "cat",
  "description": "A cute cat familiar",
  "art": [
    "  /\\_/\\  ",
    " ( @@ ) ",
    " =( Y )=",
    "   ^ ^  "
  ],
  "anchor": {"x": 4, "y": 0},
  "eyes": {
    "line": 1,
    "col": 3,
    "width": 2,
    "placeholder": "@@"
  },
  "mouth": {
    "line": 2,
    "col": 4,
    "width": 1,
    "placeholder": "Y"
  },
  "colors": {
    "eyes": "#7CFC00",
    "mouth": "#FF69B4"
  }
}
```

## Character Color Customization

You can customize character colors using the color flags:

```bash
# Custom colored cat
familiar-says --character cat --eye-color "#00FF00" --mouth-color pink "Custom colors!"

# Full color customization
familiar-says --character dragon --outline-color fire --eye-color gold --mouth-color red "Fiery dragon!"

# See all available named colors
familiar-says --list-colors
```

Supported color formats:
- **Hex codes**: `#FF6B6B`, `#F6B`, `FF6B6B`
- **ANSI 256**: `196`, `82`, `46`
- **Named colors**: `red`, `blue`, `pink`, `fire`, `ocean`, `sunset`, etc.

## Examples

### Combine features:

```bash
# Happy cat with rainbow effect
familiar-says --character cat --mood happy --effect rainbow "I love rainbows!"

# Animated angry cyber-themed message
familiar-says --mood angry --theme cyber --animate --speed 30 "System breach detected!"

# Thought bubble with sparkles
familiar-says --think --effect sparkle "What should I think about?"

# Custom width for long messages
familiar-says --width 60 "This is a much longer message that needs more space to display properly without wrapping too much."

# Custom colored owl
familiar-says --character owl --eye-color gold --mood wise "Hoot hoot!"
```

### Piping input:

```bash
echo "Hello from stdin!" | familiar-says

fortune | familiar-says --mood happy --theme rainbow
```

## Architecture

The project is organized into several packages:

One-shot rendering:

- `internal/bubble` - Speech bubble generation with word wrapping
- `internal/personality` - Theme and mood system
- `internal/animation` - Terminal animations using Bubble Tea
- `internal/effects` - Visual effects engine
- `internal/character` - Character rendering engine (loads JSON character files)
- `internal/canvas` - Low-level character rendering with color support and composition
- `internal/config` - Configuration file, profile, and environment variable management
- `internal/errors` - Custom error types with consistent formatting
- `cmd` - CLI application using Cobra
- `characters/` - Built-in character definitions in JSON format

The live familiar:

- `internal/stage` - Real-time rendering: a paint-per-cell grid that coalesces styled runs, particles, typewriter speech bubbles, scenery
- `internal/creature` - The familiar's body: movement, moods, blinking, naps, choreographed reactions, sprite rendering from the character JSON
- `internal/brain` - The familiar's mind: a rule engine that turns shell events into suggestions and quips (typo correction, exit codes, project and git awareness, memory, the optional oracle)
- `internal/ipc` - The Unix-socket event protocol between shells and a running familiar
- `internal/live` - The world that ties those together, and its two front-ends: the full-screen companion and the headless prompt perch
- `internal/shell` - The generated fish, bash and zsh integration scripts
- `internal/wrap` - The pass-through terminal that lets the familiar read command output and window focus

```
 shell hooks ──events──▶ ipc ──▶ live.World ──▶ brain  (what does this mean? what should I say?)
 (fish/bash/zsh)                    │   ▲          │
                                    │   └─reactions┘
                                    ▼
                    creature + particles + bubble ──▶ stage.Grid ──▶ TUI pane, or frame file ──▶ your prompt
```

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
