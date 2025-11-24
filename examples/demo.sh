#!/bin/bash
# Example demonstrations of familiar-says capabilities

echo "=== Welcome to familiar-says demo ==="
echo ""

# Build first
echo "Building familiar-says..."
make build > /dev/null 2>&1
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
./familiar-says --character characters/cat.cow "Meow!"
echo ""
sleep 2

echo "5. Using the dragon character:"
./familiar-says --character characters/dragon.cow --mood excited "Rawr!"
echo ""
sleep 2

echo "6. Using the penguin character:"
./familiar-says --character characters/penguin.cow "Tux says hi!"
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
./familiar-says --character characters/robot.cow --theme retro --mood excited --effect rainbow "All features activated!"
echo ""

echo "=== Demo complete! ==="
echo ""
echo "Try it yourself:"
echo "  ./familiar-says --help"
echo "  ./familiar-says --list-themes"
echo "  ./familiar-says --list-moods"
echo "  ./familiar-says --list-effects"
