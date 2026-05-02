#!/bin/bash
# scripts/migrate-existing-docs.sh
# One-time script to migrate existing documentation to new structure

set -e

echo "📚 Starting documentation migration..."
echo

cd "$(dirname "$0")/.."

VAULT="hourglass-vault"
LEGACY="$VAULT/LEGACY"

# Check if LEGACY folder exists
if [ ! -d "$LEGACY" ]; then
    echo "❌ LEGACY folder not found. Has the vault been set up?"
    exit 1
fi

echo "Found legacy docs in $LEGACY"
echo

# Migration mapping (manual review needed)
echo "📋 Suggested migration plan:"
echo

# Count files in each category
AUTH_DOCS=$(ls -1 $LEGACY | grep -iE "(auth|login|password)" | wc -l | tr -d ' ')
BACKEND_DOCS=$(ls -1 $LEGACY | grep -iE "(backend|handler|api)" | wc -l | tr -d ' ')
FRONTEND_DOCS=$(ls -1 $LEGACY | grep -iE "(frontend|react|component)" | wc -l | tr -d ' ')
FEATURE_DOCS=$(ls -1 $LEGACY | grep -iE "(time|expense|contract|project|organization|user)" | wc -l | tr -d ' ')
SCHEMA_DOCS=$(ls -1 $LEGACY | grep -iE "(schema|database|migration)" | wc -l | tr -d ' ')
OPS_DOCS=$(ls -1 $LEGACY | grep -iE "(deploy|test|setup|dev)" | wc -l | tr -d ' ')

echo "Legacy files by category:"
echo "  Auth-related: $AUTH_DOCS"
echo "  Backend patterns: $BACKEND_DOCS"
echo "  Frontend: $FRONTEND_DOCS"
echo "  Features: $FEATURE_DOCS"
echo "  Schema: $SCHEMA_DOCS"
echo "  Operations: $OPS_DOCS"
echo

echo "ℹ️  Manual steps required:"
echo "   1. Review files in $LEGACY"
echo "   2. Migrate to 01-Features/ (user-facing functionality)"
echo "   3. Migrate to 02-Technical/ (implementation guides)"
echo "   4. Migrate to 03-Schema/ (design & contracts)"
echo "   5. Keep remaining in LEGACY/ for reference"
echo

echo "💡 Recommended first migrations:"
echo "   - 05-Auth-System.md → Start of F04-User-Authentication.md"
echo "   - 04-Backend-Patterns.md → Enhance T01-Hexagonal-Architecture.md"
echo "   - 03-Database-Schema.md → Basis for S01-Database-ERD.md"
echo

echo "✅ Migration helper ready!"
echo "   Run this script again after manual review to verify completeness."
