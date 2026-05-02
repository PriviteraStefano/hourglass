#!/bin/bash
# scripts/docs-check.sh
# Documentation completeness checker

echo "📊 Documentation Completeness Report"
echo "===================================="
echo

cd "$(dirname "$0")/.."

# Count documents
FEATURES=$(find hourglass-vault/01-Features -name "*.md" ! -name "*-draft-*" 2>/dev/null | wc -l | tr -d ' ')
TECHNICAL=$(find hourglass-vault/02-Technical -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
SCHEMA=$(find hourglass-vault/03-Schema -name "*.md" 2>/dev/null | wc -l | tr -d ' ')

echo "📖 FEATURES: $FEATURES documents"
echo "🔧 TECHNICAL: $TECHNICAL documents"
echo "🏗️ SCHEMA: $SCHEMA documents"
echo

# Find draft docs
DRAFTS=$(find hourglass-vault -name "*-draft-*.md" 2>/dev/null)
if [ -n "$DRAFTS" ]; then
    echo "⚠️  Pending drafts:"
    echo "$DRAFTS"
    echo
fi

# Check for TODOs
TODOS=$(grep -r "\- \[ \]" hourglass-vault/ --include="*.md" 2>/dev/null | wc -l | tr -d ' ')
if [ "$TODOS" -gt 0 ]; then
    echo "📝 Unchecked TODOs: $TODOS"
    echo
fi

# Mermaid diagram count
MERMAID=$(grep -r "^\`\`\`mermaid" hourglass-vault/ --include="*.md" 2>/dev/null | wc -l | tr -d ' ')
echo "📊 Mermaid diagrams: $MERMAID"
echo

# Last updated check (last 7 days)
RECENT=$(find hourglass-vault -name "*.md" -mtime -7 2>/dev/null | wc -l | tr -d ' ')
echo "📅 Recently updated (last 7 days): $RECENT documents"
if [ "$RECENT" -gt 0 ]; then
    find hourglass-vault -name "*.md" -mtime -7 -exec basename {} \; 2>/dev/null | head -10
fi
echo

# Completeness percentage
if [ "$FEATURES" -gt 0 ]; then
    echo "ℹ️  Documentation ratio:"
    if [ "$FEATURES" -gt 0 ]; then
        TECH_RATIO=$((TECHNICAL * 100 / FEATURES))
        echo "   Technical/Features: ${TECH_RATIO}%"
    fi
    if [ "$FEATURES" -gt 0 ]; then
        SCHEMA_RATIO=$((SCHEMA * 100 / FEATURES))
        echo "   Schema/Features: ${SCHEMA_RATIO}%"
    fi
fi
echo

# Warnings
WARNINGS=0
if [ "$FEATURES" -eq 0 ]; then
    echo "⚠️  WARNING: No feature documents found!"
    WARNINGS=$((WARNINGS + 1))
fi

if [ "$MERMAID" -lt "$FEATURES" ]; then
    echo "⚠️  WARNING: Some features may be missing Mermaid diagrams"
    WARNINGS=$((WARNINGS + 1))
fi

if [ "$WARNINGS" -gt 0 ]; then
    echo
    echo "❌ Total warnings: $WARNINGS"
    exit 1
else
    echo "✅ All checks passed!"
    exit 0
fi
