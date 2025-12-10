#!/bin/bash
# Demo script for familiar-says advanced bubble types
# Run this to see all the new bubble styles in action

echo "=== Advanced Bubble Types Demo ==="
echo ""

echo "1. Standard Say Bubble:"
./familiar-says --bubble-style say "Hello, I'm a regular speech bubble!"
echo ""

echo "2. Think Bubble (with 'o' connector):"
./familiar-says --bubble-style think "Hmm, what should I do today?"
echo ""

echo "3. Shout Bubble (jagged edges):"
./familiar-says --bubble-style shout "THIS IS IMPORTANT"
echo ""

echo "4. Whisper Bubble (dotted borders):"
./familiar-says --bubble-style whisper "shh, keep it down"
echo ""

echo "5. Song Bubble (musical notes):"
./familiar-says --bubble-style song "Do re mi fa so la ti do"
echo ""

echo "6. Code Bubble (box borders):"
./familiar-says --bubble-style code 'fmt.Println("Hello, World!")'
echo ""

echo "=== Tail Direction Demo ==="
echo ""

echo "7. Tail pointing down (default):"
./familiar-says --tail-direction down "Looking down"
echo ""

echo "8. Tail pointing up:"
./familiar-says --tail-direction up "Looking up"
echo ""

echo "9. Different characters with different styles:"
./familiar-says --character cat --bubble-style song "Meow meow meow"
echo ""
./familiar-says --character dragon --bubble-style shout "ROAR"
echo ""
./familiar-says --character owl --bubble-style think "Who?"
echo ""

echo "=== Backward Compatibility ==="
echo ""

echo "10. Using --think flag (deprecated but still works):"
./familiar-says --think "This still works!"
echo ""

echo "=== Combined with other features ==="
echo ""

echo "11. Shout with rainbow effect:"
./familiar-says --bubble-style shout --effect rainbow-text "RAINBOW SHOUT"
echo ""

echo "Demo complete!"
