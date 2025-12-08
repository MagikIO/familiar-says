#!/bin/bash
# Example demonstrations of familiar-says capabilities

echo "=== Welcome to familiar-says demo ==="
echo ""

# Build first (run from project root)
echo "Building familiar-says..."
cd "$(dirname "$0")/.." || exit 1
make build
echo ""

# Basic usage
echo "1. Basic usage:"
./familiar-says "Hello, World!"
echo ""
sleep 2

# Different themes
echo "2. Rainbow theme with happy mood:"
./familiar-says --theme rainbow --mood happy "I'm feeling colorful!"
echo ""
sleep 2

# Cyber theme
echo "3. Cyber theme with angry mood:"
./familiar-says --theme cyber --mood angry "Access denied!"
echo ""
sleep 2

# Different characters
echo "4. Using the cat character:"
./familiar-says --character cat "Meow!"
echo ""
sleep 2

echo "5. Using the dragon character:"
./familiar-says --character dragon --mood excited "Rawr!"
echo ""
sleep 2

echo "6. Using the penguin character:"
./familiar-says --character penguin "Tux says hi!"
echo ""
sleep 2

# Effects
echo "7. With sparkle effect:"
./familiar-says --effect sparkle "Sparkles!"
echo ""
sleep 2

# Thought bubble
echo "8. Thought bubble:"
./familiar-says --think --mood sleepy "I wonder..."
echo ""
sleep 2

# Long message with custom width
echo "9. Long message with custom width:"
./familiar-says --width 60 "This is a much longer message that demonstrates how the word wrapping works when you need to display more text than can fit on a single line."
echo ""
sleep 2

# All features combined
echo "10. Combining multiple features:"
./familiar-says --character robot --theme retro --mood excited --effect rainbow "All features activated!"
echo ""
sleep 2

# Rainbow text only
echo "11. Rainbow text effect (colors only message, not character):"
./familiar-says --character cat --effect rainbow-text "Colorful message, plain character!"
echo ""
sleep 2

# Character color customization
echo "12. Custom character colors (eye color):"
./familiar-says --character dragon --eye-color fire "My eyes burn with passion!"
echo ""
sleep 2

echo "13. Custom character colors (multiple parts):"
./familiar-says --character cat --outline-color pink --eye-color cyan --mouth-color red "Fully customized!"
echo ""
sleep 2

echo "14. Using hex color codes:"
./familiar-says --character robot --outline-color "#00FF00" --eye-color "#FF00FF" "Matrix mode activated!"
echo ""
sleep 2

echo "15. Using ANSI color numbers:"
./familiar-says --character owl --eye-color 226 "Using ANSI 256 colors!"
echo ""
sleep 2

echo "=== Demo complete! ==="
echo ""
echo "Try it yourself:"
echo "  ./familiar-says --help"
echo "  ./familiar-says --list-themes"
echo "  ./familiar-says --list-moods"
echo "  ./familiar-says --list-effects"
echo "  ./familiar-says --list-colors"
echo ""
echo "Character color customization:"
echo "  ./familiar-says --character cat --eye-color green \"Green eyes!\""
echo "  ./familiar-says --character dragon --outline-color fire --eye-color gold \"Fiery dragon!\""
