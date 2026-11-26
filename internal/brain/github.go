package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/creature"
)

// PullRequest is the part of a GitHub pull request the familiar cares about.
type PullRequest struct {
	Number int
	Title  string
	Checks string   // "pass", "fail", "pending", or "" when there are none
	Failed []string // names of the checks that failed
	Review string   // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED or ""
}

// GitHubStatus is one look at a repository's pull requests.
type GitHubStatus struct {
	Root      string
	Current   *PullRequest  // the pull request for the checked-out branch
	Reviews   []PullRequest // pull requests waiting on the user's review
	Unfocused bool
	Err       error
}

// runGH runs the gh CLI in dir. It is a variable so tests can stand in for it.
var runGH = func(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "NO_COLOR=1")
	return cmd.Output()
}

// ScoutGitHub asks GitHub, through the gh CLI, how the pull requests of the
// repository at root are doing. It makes a network request, so it only ever
// runs when the user has turned GitHub awareness on.
func ScoutGitHub(root, focusFile string) GitHubStatus {
	st := GitHubStatus{Root: root}
	out, err := runGH(root, "pr", "status", "--json", "number,title,reviewDecision,statusCheckRollup")
	if err != nil {
		st.Err = err
		return st
	}
	var raw struct {
		CurrentBranch *ghPR  `json:"currentBranch"`
		NeedsReview   []ghPR `json:"needsReview"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		st.Err = err
		return st
	}
	if raw.CurrentBranch != nil && raw.CurrentBranch.Number != 0 {
		pr := raw.CurrentBranch.summarize()
		st.Current = &pr
	}
	for _, pr := range raw.NeedsReview {
		st.Reviews = append(st.Reviews, pr.summarize())
	}
	if focusFile != "" {
		if data, err := os.ReadFile(focusFile); err == nil {
			st.Unfocused = strings.TrimSpace(string(data)) == "0"
		}
	}
	return st
}

type ghPR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	ReviewDecision string `json:"reviewDecision"`
	Checks         []struct {
		Name       string `json:"name"`       // check runs
		Context    string `json:"context"`    // commit statuses
		Status     string `json:"status"`     // QUEUED, IN_PROGRESS, COMPLETED
		Conclusion string `json:"conclusion"` // SUCCESS, FAILURE, ...
		State      string `json:"state"`      // commit statuses: SUCCESS, FAILURE, PENDING, ERROR
	} `json:"statusCheckRollup"`
}

func (p ghPR) summarize() PullRequest {
	pr := PullRequest{Number: p.Number, Title: p.Title, Review: p.ReviewDecision}
	pending := false
	for _, c := range p.Checks {
		name := c.Name
		if name == "" {
			name = c.Context
		}
		switch strings.ToUpper(c.Conclusion + c.State) {
		case "FAILURE", "TIMED_OUT", "ERROR", "STARTUP_FAILURE", "ACTION_REQUIRED":
			pr.Failed = append(pr.Failed, name)
		case "SUCCESS", "NEUTRAL", "SKIPPED", "CANCELLED":
		default:
			pending = true
		}
	}
	sort.Strings(pr.Failed)
	switch {
	case len(pr.Failed) > 0:
		pr.Checks = "fail"
	case pending:
		pr.Checks = "pending"
	case len(p.Checks) > 0:
		pr.Checks = "pass"
	}
	return pr
}

// ghMemory is what was true of a repository the last time it was checked.
type ghMemory struct {
	checks, review string
	number         int
	waiting        map[int]bool
	backoff        time.Time
}

// pollGitHub schedules a look at GitHub when one is due.
func (b *Brain) pollGitHub(now time.Time) {
	if !b.cfg.GitHub || b.gitRoot == "" || b.ghBusy || now.Before(b.ghNext) {
		return
	}
	if m := b.gh[b.gitRoot]; m != nil && now.Before(m.backoff) {
		return
	}
	// Only while there is someone here to tell.
	if b.lastEvent.IsZero() || now.Sub(b.lastEvent) > 20*time.Minute {
		return
	}
	b.ghBusy = true
	b.ghNext = now.Add(5 * time.Minute)
	if now.Before(b.ghWatch) {
		b.ghNext = now.Add(30 * time.Second) // CI is running: look more often
	}
	root, focus := b.gitRoot, b.focusFile
	b.tasks = append(b.tasks, func() any { return ScoutGitHub(root, focus) })
}

// onGitHub compares a fresh look at GitHub with the last one and speaks up
// about what changed.
func (b *Brain) onGitHub(st GitHubStatus) []Reaction {
	b.ghBusy = false
	now := b.now()
	m := b.gh[st.Root]
	first := m == nil
	if first {
		m = &ghMemory{waiting: map[int]bool{}}
		b.gh[st.Root] = m
	}
	if st.Err != nil {
		// Not a GitHub repository, not signed in, or offline. Don't nag,
		// and don't keep trying.
		m.backoff = now.Add(30 * time.Minute)
		return nil
	}

	var rs []Reaction
	say := func(r Reaction, note string) {
		r.Must, r.Sticky = true, true
		if b.cfg.Notify == "always" || (b.cfg.Notify == "unfocused" && st.Unfocused) {
			r.Notify = note
		}
		rs = append(rs, r)
	}

	if pr := st.Current; pr != nil {
		tag := fmt.Sprintf("#%d", pr.Number)
		samePR := pr.Number == m.number
		switch {
		case pr.Checks == "fail" && (!samePR || m.checks != "fail"):
			what := strings.Join(pr.Failed[:min(2, len(pr.Failed))], ", ")
			say(Reaction{Text: fmt.Sprintf("CI failed on %s: %s.", tag, clip(what, 60)), Suggest: "gh pr checks", Act: creature.ActWince},
				fmt.Sprintf("CI failed on %s (%s)", tag, clip(pr.Title, 50)))
		case pr.Checks == "pass" && samePR && m.checks == "pending":
			say(Reaction{Text: fmt.Sprintf("CI is green on %s.", tag), Act: creature.ActCheer},
				fmt.Sprintf("CI passed on %s (%s)", tag, clip(pr.Title, 50)))
		case pr.Checks == "pass" && samePR && m.checks == "fail":
			say(Reaction{Text: fmt.Sprintf("CI is green again on %s.", tag), Act: creature.ActCelebrate},
				fmt.Sprintf("CI passed on %s (%s)", tag, clip(pr.Title, 50)))
		}
		if samePR && pr.Review != m.review {
			switch pr.Review {
			case "APPROVED":
				say(Reaction{Text: fmt.Sprintf("%s was approved!", tag), Act: creature.ActCelebrate}, tag+" was approved")
			case "CHANGES_REQUESTED":
				say(Reaction{Text: fmt.Sprintf("Changes were requested on %s.", tag), Suggest: "gh pr view --comments", Act: creature.ActAlert},
					"Changes requested on "+tag)
			}
		}
		if pr.Checks == "pending" {
			b.ghWatch = now.Add(20 * time.Minute)
			b.ghNext = now.Add(30 * time.Second)
		}
		m.number, m.checks, m.review = pr.Number, pr.Checks, pr.Review
	} else {
		m.number, m.checks, m.review = 0, "", ""
	}

	var fresh []PullRequest
	waiting := map[int]bool{}
	for _, pr := range st.Reviews {
		waiting[pr.Number] = true
		if !m.waiting[pr.Number] {
			fresh = append(fresh, pr)
		}
	}
	m.waiting = waiting
	if len(fresh) > 0 {
		text := fmt.Sprintf("#%d is waiting on your review: %s.", fresh[0].Number, clip(fresh[0].Title, 50))
		if len(fresh) > 1 {
			text = fmt.Sprintf("%d pull requests are waiting on your review, starting with #%d: %s.", len(fresh), fresh[0].Number, clip(fresh[0].Title, 40))
		}
		say(Reaction{Text: text, Suggest: fmt.Sprintf("gh pr view %d --web", fresh[0].Number), Act: creature.ActAlert},
			fmt.Sprintf("Review requested: #%d %s", fresh[0].Number, clip(fresh[0].Title, 50)))
	}
	return rs
}
