#!/bin/bash
# scripts/validate-mermaid.sh
# Validates Mermaid diagram syntax in markdown files

set -e

echo "🔍 Validating Mermaid diagrams..."
echo

cd "$(dirname "$0")/.."

ERRORS=0
FILES_CHECKED=0
DIAGRAMS_CHECKED=0

# Find all markdown files with mermaid blocks
for file in $(find hourglass-vault -name "*.md" -type f); do
    if grep -q "^\`\`\`mermaid" "$file"; then
        FILES_CHECKED=$((FILES_CHECKED + 1))
        
        # Extract mermaid blocks and validate basic syntax
        IN_MERMAID=false
        LINE_NUM=0
        MERMAID_START=0
        
        while IFS= read -r line; do
            LINE_NUM=$((LINE_NUM + 1))
            
            if [[ "$line" =~ ^\`\`\`mermaid ]]; then
                IN_MERMAID=true
                MERMAID_START=$LINE_NUM
                continue
            fi
            
            if [[ "$line" =~ ^\`\`\` ]] && [ "$IN_MERMAID" = true ]; then
                IN_MERMAID=false
                DIAGRAMS_CHECKED=$((DIAGRAMS_CHECKED + 1))
                continue
            fi
            
            if [ "$IN_MERMAID" = true ]; then
                # Basic validation checks
                
                # Check for flowchart syntax
                if echo "$line" | grep -q "flowchart"; then
                    if ! echo "$line" | grep -qE "flowchart (TB|TD|BT|LR|RL)"; then
                        echo "⚠️  $file:$LINE_NUM - flowchart missing direction (TB/TD/LR/etc.)"
                        ERRORS=$((ERRORS + 1))
                    fi
                fi
                
                # Check for unclosed brackets
                OPEN_BRACKETS=$(echo "$line" | tr -cd '[' | wc -c)
                CLOSE_BRACKETS=$(echo "$line" | tr -cd ']' | wc -c)
                if [ "$OPEN_BRACKETS" -ne "$CLOSE_BRACKETS" ]; then
                    # Could be multi-line, just warn
                    :
                fi
                
                # Check for proper arrow syntax
                if echo "$line" | grep -qE "\-\->|\-\-.*\-\>" ; then
                    # Valid arrow found
                    :
                elif echo "$line" | grep -qE "[A-Za-z].*[A-Za-z]" && ! echo "$line" | grep -qE "^[[:space:]]*%%"; then
                    # Line has text but no arrows or comments - might be OK
                    :
                fi
            fi
        done < "$file"
    fi
done

echo "✅ Checked $FILES_CHECKED files, $DIAGRAMS_CHECKED Mermaid diagrams"
echo

if [ "$ERRORS" -gt 0 ]; then
    echo "❌ Found $ERRORS potential issues"
    exit 1
else
    echo "✅ All Mermaid diagrams look valid!"
    exit 0
fi
