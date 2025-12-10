package canvas

// testCatCharacter returns a test cat character for testing purposes.
// This provides a consistent character for unit tests without relying on embedded FS.
func testCatCharacter() *Character {
	return &Character{
		Name:        "cat",
		Description: "A cute cat familiar",
		Art: []string{
			`  /\_/\  `,
			` ( @@ ) `,
			` =( Y )=`,
			`   ^ ^  `,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
		Mouth: &Slot{
			Line:        2,
			Col:         4,
			Width:       1,
			Placeholder: "Y",
		},
	}
}

// testOwlCharacter returns a test owl character for testing purposes.
func testOwlCharacter() *Character {
	return &Character{
		Name:        "owl",
		Description: "A wise owl familiar",
		Art: []string{
			`  ,_,  `,
			` (@@)  `,
			` /)_)  `,
			`  ""   `,
		},
		Anchor: Anchor{X: 3, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         2,
			Width:       2,
			Placeholder: "@@",
		},
	}
}
