package brain

// lines holds everything the familiar says unprompted, by occasion. A %s is
// filled in by the rule that uses the line; see brain.go.
var lines = map[string][]string{
	"morning":   {"Morning. Coffee first, then commits.", "Good morning! What are we building?", "Up early. I like it."},
	"afternoon": {"Afternoon. Let's get into it.", "Hello again. Where were we?", "Good afternoon. I'll keep watch."},
	"evening":   {"Evening shift. I'm here.", "Good evening. One more thing before bed?", "Evening. The quiet hours are the good ones."},
	"night":     {"Still up? Me too, apparently.", "Night owl hours. I'll keep you company.", "It's late. I won't tell anyone."},

	"back":      {"Welcome back.", "Oh! You're back.", "There you are. I kept your spot warm."},
	"back_fond": {"You're back! I missed you.", "There you are! It's dull in here without you.", "Oh good, it's you. I was getting lonely."},
	"dress":     {"How do I look?", "Oh, I like this one.", "Fancy. Do I look fancy?"},
	"undress":   {"Back to basics.", "Au naturel it is."},
	"woke":      {"Hm? I'm up, I'm up.", "*yawn* What did I miss?", "I wasn't sleeping. I was resting my eyes."},
	"pet":       {"Hehe.", "That's the spot.", "Again, please.", "I suppose you may continue.", "Best part of my day."},

	"test_pass":   {"Green across the board.", "Tests pass. Ship it?", "All green. Nicely done.", "They pass! Don't touch anything."},
	"test_fail":   {"Red. The first failure is the one to read.", "Tests disagree with you.", "Not yet. Closer, though?", "A failing test is just a test doing its job."},
	"build_fail":  {"It doesn't build. Top error first.", "The compiler has notes.", "Build's broken. Probably one small thing."},
	"recover":     {"There it is!", "Fixed. That one fought back.", "And it works. Told you.", "Persistence pays."},
	"recover_big": {"After %d tries. Never doubted you.", "%d failures, then that. Beautiful.", "Got there on attempt %d. Worth it."},

	"long_done": {"Done. That took %s.", "Finished after %s.", "%s later: all done.", "That's finished. %s, if you were wondering."},
	"long_fail": {"%s, and then it failed. Oof.", "It ran for %s before giving up.", "%s of waiting for an error. Sorry."},

	"fail":      {"Hm. That didn't take.", "Exit %d. Want to look closer?", "Not quite.", "That one bounced."},
	"fail_many": {"%d in a row. `%s --help` might save a step.", "Attempt %d. Maybe read the error out loud?", "%d tries. Step back; what changed last?"},
	"interrupt": {"Bailed out. Fair.", "Ctrl-C, the universal undo.", "Changed your mind?"},

	"streak": {"%d clean commands in a row.", "%d without an error. You're in a groove.", "Streak: %d. Don't jinx it."},
	"commit": {"Committed.", "Another one in the history books.", "Saved for posterity."},
	"push":   {"Shipped.", "Off it goes.", "Pushed. No take-backs."},
	"friday": {"Pushing on a Friday afternoon. Bold.", "Friday deploy. I admire the confidence."},
	"danger": {"Careful with that one.", "I'm not looking.", "You're sure? You look sure.", "Deep breath."},
	"sudo":   {"With great power...", "Ooh, the big hammer."},
	"relief": {"Phew.", "Still in one piece."},

	"break":     {"You've been at it for %s. Stand up, look at something far away.", "%s of solid work. Water? Stretch?", "It's been %s. Your shoulders called."},
	"late":      {"It's %s. The bug will still be there tomorrow.", "%s already. Past-midnight code needs daytime review.", "It's %s. Just saying."},
	"think":     {"Let me think...", "Hmm. One moment.", "Consulting the ancient texts..."},
	"no_oracle": {"I'd need an oracle for that. Set live.oracle in the config and I can ask one."},

	"musing": {
		"I wonder where the cursor goes when it blinks.",
		"Nothing is on fire. I checked.",
		"Do you ever just watch the prompt?",
		"I like it here.",
		"If a command fails and nobody reads the error, did it fail?",
		"Still here. Just so you know.",
		"I counted the columns. All present.",
		"Quiet in here.",
	},

	"tip": {
		"`cd -` takes you back to wherever you just were.",
		"Ctrl-R searches your history. Faster than the up arrow.",
		"`git switch -` flips back to the previous branch.",
		"`git commit --fixup <sha>` plus `rebase --autosquash` tidies history painlessly.",
		"`git log --oneline --graph` is the history view worth an alias.",
		"`git stash -u` also stashes untracked files.",
		"`!!` reruns the last command. `sudo !!` reruns it with feeling.",
		"`less +F file` follows a file like `tail -f`, and Ctrl-C lets you scroll.",
		"`ssh -J jump host` hops through a bastion in one go.",
		"`curl -sS` is quiet but still shows errors.",
		"`git diff --staged` shows what you're about to commit.",
		"`git restore -p` lets you discard changes hunk by hunk.",
		"`du -sh * | sort -h` finds what's eating the disk.",
		"`git worktree add ../x branch` gives you a second checkout without stashing.",
	},
	"tip_fish": {
		"Alt-S puts `sudo` in front of the current command.",
		"Alt-E opens the command line in your editor.",
		"Alt-Left and Alt-Right walk your directory history.",
		"`abbr` beats `alias` here: it expands as you type, so history stays honest.",
		"Type part of a command, then Up, to search history for it.",
		"Alt-L lists the directory under the cursor.",
		"`funced name` edits a function live; `funcsave` keeps it.",
		"Ctrl-X copies the command line to the clipboard.",
	},
	"tip_bash": {
		"Alt-. pastes the last argument of the previous command.",
		"`cd -` and `pushd`/`popd` save a lot of typing.",
		"Ctrl-X Ctrl-E opens the command line in your editor.",
		"`^old^new` reruns the last command with a substitution.",
	},
}
