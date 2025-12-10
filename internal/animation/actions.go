package animation

import "strings"

// Action represents a character action/animation type.
type Action string

// Standard action types available for characters.
const (
	ActionNone      Action = "none"
	ActionIdle      Action = "idle"
	ActionBlink     Action = "blink"
	ActionWave      Action = "wave"
	ActionJump      Action = "jump"
	ActionTailWag   Action = "tail_wag"
	ActionNod       Action = "nod"
	ActionBreathe   Action = "breathe"
	ActionHeadTilt  Action = "head_tilt"
	ActionEarWiggle Action = "ear_wiggle"
	ActionHop       Action = "hop"
	ActionWaddle    Action = "waddle"
	ActionWingFlap  Action = "wing_flap"
	ActionHeadBob   Action = "head_bob"
)

// actionDescriptions maps actions to their descriptions.
var actionDescriptions = map[Action]string{
	ActionNone:      "No animation (static)",
	ActionIdle:      "Default idle animation (usually blink)",
	ActionBlink:     "Periodic eye blinking",
	ActionWave:      "Wave gesture",
	ActionJump:      "Jump motion",
	ActionTailWag:   "Tail wagging animation",
	ActionNod:       "Head nodding",
	ActionBreathe:   "Subtle breathing effect",
	ActionHeadTilt:  "Tilting head side to side",
	ActionEarWiggle: "Ear wiggling motion",
	ActionHop:       "Small hopping motion",
	ActionWaddle:    "Waddling walk motion",
	ActionWingFlap:  "Wing flapping animation",
	ActionHeadBob:   "Head bobbing up and down",
}

// AllActions returns a slice of all available actions.
func AllActions() []Action {
	return []Action{
		ActionNone,
		ActionIdle,
		ActionBlink,
		ActionWave,
		ActionJump,
		ActionTailWag,
		ActionNod,
		ActionBreathe,
		ActionHeadTilt,
		ActionEarWiggle,
		ActionHop,
		ActionWaddle,
		ActionWingFlap,
		ActionHeadBob,
	}
}

// GetActionDescription returns a human-readable description of an action.
func GetActionDescription(action Action) string {
	if desc, ok := actionDescriptions[action]; ok {
		return desc
	}
	return "Unknown action"
}

// ValidateAction checks if a string is a valid action name.
func ValidateAction(name string) bool {
	action := Action(strings.ToLower(name))
	_, ok := actionDescriptions[action]
	return ok
}

// ParseAction converts a string to an Action, returning ActionNone if invalid.
func ParseAction(name string) Action {
	action := Action(strings.ToLower(name))
	if _, ok := actionDescriptions[action]; ok {
		return action
	}
	return ActionNone
}

// IsIdleAction returns true if the action is typically used for idle animations.
func IsIdleAction(action Action) bool {
	switch action {
	case ActionIdle, ActionBlink, ActionBreathe, ActionTailWag:
		return true
	default:
		return false
	}
}

// IsTriggeredAction returns true if the action is typically triggered once.
func IsTriggeredAction(action Action) bool {
	switch action {
	case ActionWave, ActionJump, ActionNod, ActionHop:
		return true
	default:
		return false
	}
}
