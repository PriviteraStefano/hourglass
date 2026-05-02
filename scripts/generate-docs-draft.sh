#!/bin/bash
# scripts/generate-docs-draft.sh
# Generates documentation draft from GitHub PR

set -e

PR_NUMBER=$1
if [ -z "$PR_NUMBER" ]; then
    echo "Usage: $0 <pr-number>"
    exit 1
fi

echo "📝 Generating documentation draft for PR #$PR_NUMBER"

# Fetch PR details
PR_TITLE=$(gh pr view $PR_NUMBER --json title -q .title)
PR_BODY=$(gh pr view $PR_NUMBER --json body -q .body)
PR_AUTHOR=$(gh pr view $PR_NUMBER --json author -q .author.login)
MERGE_DATE=$(date +%Y-%m-%d)

# Get changed files
CHANGED_FILES=$(gh pr view $PR_NUMBER --json files -q '.[].filename')

# Detect feature areas
FEATURE_AREAS=""
if echo "$CHANGED_FILES" | grep -q "internal/core"; then
    FEATURE_AREAS="$FEATURE_AREAS hexagonal-architecture"
fi
if echo "$CHANGED_FILES" | grep -q "internal/handlers"; then
    FEATURE_AREAS="$FEATURE_AREAS handlers"
fi
if echo "$CHANGED_FILES" | grep -q "web/src"; then
    FEATURE_AREAS="$FEATURE_AREAS frontend"
fi
if echo "$CHANGED_FILES" | grep -q "schema/"; then
    FEATURE_AREAS="$FEATURE_AREAS database-schema"
fi

# Create draft filename
SAFE_TITLE=$(echo $PR_TITLE | tr ' ' '-' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]//g' | cut -c1-50)
DRAFT_FILE="hourglass-vault/01-Features/F-draft-$PR_NUMBER-$SAFE_TITLE.md"

# Create draft
cat > "$DRAFT_FILE" << EOF
# Feature: $PR_TITLE

## PR Reference
- **PR**: #$PR_NUMBER
- **Merged**: $MERGE_DATE
- **Author**: @$PR_AUTHOR
- **Areas**:$FEATURE_AREAS

## Description
$PR_BODY

## TODO: Step 1 - Feature Documentation
- [ ] Document user stories
- [ ] Add user workflows with Mermaid diagrams
- [ ] List acceptance criteria met

## TODO: Step 2 - Technical Documentation
- [ ] Document backend changes
- [ ] Document frontend changes
- [ ] Add code examples
- [ ] Document testing approach

## TODO: Step 3 - Schema Documentation
- [ ] Update domain models
- [ ] Update database schema
- [ ] Update API contracts
- [ ] Add state machine diagrams

---
*Auto-generated draft. Complete the checklists above.*
EOF

echo "✅ Draft created: $DRAFT_FILE"
echo
echo "📋 Next steps:"
echo "   1. Complete Step 1 checklist (Feature doc)"
echo "   2. Complete Step 2 checklist (Technical doc)"
echo "   3. Complete Step 3 checklist (Schema doc)"
echo "   4. Move to appropriate folder when complete"
echo
echo "💡 Tip: Use the templates in hourglass-vault/01-Features/_TEMPLATE.md"
