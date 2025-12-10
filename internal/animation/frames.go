package animation

import (
	"time"

	"github.com/MagikIO/familiar-says/internal/canvas"
)

// FramePlayer manages playback of character animations.
type FramePlayer struct {
	baseCharacter *canvas.Character
	animation     *canvas.AnimationSequence
	currentFrame  int
	elapsed       time.Duration
	styles        canvas.CharacterStyles
	defaultEyes   string
	defaultMouth  string
	done          bool
}

// NewFramePlayer creates a new animation player for a character.
func NewFramePlayer(char *canvas.Character, anim *canvas.AnimationSequence, styles canvas.CharacterStyles, defaultEyes, defaultMouth string) *FramePlayer {
	return &FramePlayer{
		baseCharacter: char,
		animation:     anim,
		currentFrame:  0,
		elapsed:       0,
		styles:        styles,
		defaultEyes:   defaultEyes,
		defaultMouth:  defaultMouth,
		done:          false,
	}
}

// Tick advances the animation by the given delta time and returns the current frame's canvas.
func (fp *FramePlayer) Tick(delta time.Duration) *canvas.Canvas {
	if fp.done || fp.animation == nil || len(fp.animation.Frames) == 0 {
		return fp.renderFrame(0)
	}

	fp.elapsed += delta

	// Check if we need to advance to the next frame
	currentFrameDuration := time.Duration(fp.animation.Frames[fp.currentFrame].DurationMs) * time.Millisecond
	for fp.elapsed >= currentFrameDuration {
		fp.elapsed -= currentFrameDuration
		fp.currentFrame++

		// Handle end of animation
		if fp.currentFrame >= len(fp.animation.Frames) {
			if fp.animation.Loop {
				fp.currentFrame = 0
			} else {
				fp.currentFrame = len(fp.animation.Frames) - 1
				fp.done = true
				break
			}
		}

		// Update current frame duration for next iteration
		if fp.currentFrame < len(fp.animation.Frames) {
			currentFrameDuration = time.Duration(fp.animation.Frames[fp.currentFrame].DurationMs) * time.Millisecond
		}
	}

	return fp.renderFrame(fp.currentFrame)
}

// renderFrame renders the character at the specified frame index.
func (fp *FramePlayer) renderFrame(frameIdx int) *canvas.Canvas {
	if fp.animation == nil || len(fp.animation.Frames) == 0 {
		// No animation, render with defaults
		return fp.baseCharacter.ToCanvasStyled(fp.defaultEyes, fp.defaultMouth, fp.styles)
	}

	if frameIdx < 0 || frameIdx >= len(fp.animation.Frames) {
		frameIdx = 0
	}

	frame := fp.animation.Frames[frameIdx]

	// Determine which art to use
	var charToRender *canvas.Character
	if len(frame.Art) > 0 {
		// Create a temporary character with the frame's art
		charToRender = fp.baseCharacter.Clone()
		charToRender.Art = frame.Art
	} else {
		charToRender = fp.baseCharacter
	}

	// Determine eyes and mouth expressions
	eyes := fp.defaultEyes
	if frame.Eyes != "" {
		eyes = frame.Eyes
	}

	mouth := fp.defaultMouth
	if frame.Mouth != "" {
		mouth = frame.Mouth
	}

	// Render the character
	charCanvas := charToRender.ToCanvasStyled(eyes, mouth, fp.styles)

	// Apply offsets if any
	if frame.OffsetX != 0 || frame.OffsetY != 0 {
		// Create a larger canvas to accommodate the offset
		width := charCanvas.Width + abs(frame.OffsetX)
		height := charCanvas.Height + abs(frame.OffsetY)
		offsetCanvas := canvas.NewCanvas(width, height)

		// Calculate the position with offset
		x := 0
		y := 0
		if frame.OffsetX > 0 {
			x = frame.OffsetX
		}
		if frame.OffsetY > 0 {
			y = frame.OffsetY
		}

		offsetCanvas.Overlay(charCanvas, x, y)
		return offsetCanvas
	}

	return charCanvas
}

// IsComplete returns true if a non-looping animation has finished.
func (fp *FramePlayer) IsComplete() bool {
	return fp.done
}

// Reset restarts the animation from the beginning.
func (fp *FramePlayer) Reset() {
	fp.currentFrame = 0
	fp.elapsed = 0
	fp.done = false
}

// CurrentFrameIndex returns the current frame index.
func (fp *FramePlayer) CurrentFrameIndex() int {
	return fp.currentFrame
}

// TotalFrames returns the total number of frames in the animation.
func (fp *FramePlayer) TotalFrames() int {
	if fp.animation == nil {
		return 0
	}
	return len(fp.animation.Frames)
}

// GetAnimation returns the current animation.
func (fp *FramePlayer) GetAnimation() *canvas.AnimationSequence {
	return fp.animation
}

// SetAnimation changes the current animation and resets playback.
func (fp *FramePlayer) SetAnimation(anim *canvas.AnimationSequence) {
	fp.animation = anim
	fp.Reset()
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
