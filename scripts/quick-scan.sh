#!/bin/bash
set -euo pipefail

BASE="${API_URL:-http://localhost:8080}"
PASS=0
FAIL=0
WARN=0
ERRORS=""

COOKIE_JAR=$(mktemp)
cleanup() {
  rm -f "$COOKIE_JAR"
}
trap cleanup EXIT

api() {
  local method="$1" path="$2" data="${3:-}" opts=()
  opts=(-s -S -w "%{http_code}" -c "$COOKIE_JAR" -b "$COOKIE_JAR")
  [ -n "$data" ] && opts+=(-H "Content-Type: application/json" -d "$data")
  local resp_file; resp_file=$(mktemp)
  local code
  code=$(curl "${opts[@]}" -o "$resp_file" -X "$method" "${BASE}${path}" 2>/dev/null || echo "000")
  local body; body=$(cat "$resp_file"); rm -f "$resp_file"
  echo "$code|$body"
}

expect() {
  local method="$1" path="$2" exp_code="$3" data="${4:-}" label="${5:-$method $path}"
  local result; result=$(api "$method" "$path" "$data")
  local code; code=$(echo "$result" | cut -d'|' -f1)
  local body; body=$(echo "$result" | cut -d'|' -f2-)
  if [ "$code" = "$exp_code" ]; then
    echo "  PASS  $label → $code"
    PASS=$((PASS+1))
  else
    echo "  FAIL  $label → expected $exp_code, got $code"
    FAIL=$((FAIL+1))
    ERRORS="${ERRORS}FAIL: $label (expected $exp_code, got $code)\n"
    echo "    Body: $(echo "$body" | head -c 200)"
  fi
}

expect_warn() {
  local method="$1" path="$2" exp_code="$3" data="${4:-}" label="${5:-$method $path}"
  local result; result=$(api "$method" "$path" "$data")
  local code; code=$(echo "$result" | cut -d'|' -f1)
  if [ "$code" = "$exp_code" ]; then
    echo "  PASS  $label → $code"
    PASS=$((PASS+1))
  else
    echo "  WARN  $label → expected $exp_code, got $code"
    WARN=$((WARN+1))
    echo "    Body: $(echo "$body" | head -c 200)"
  fi
}

echo "============================================"
echo "  Hourglass API Quick-Scan Probe"
echo "  Target: $BASE"
echo "  $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo "============================================"
echo ""

# Phase 1: Registration and authentication
echo "--- Auth Flow ---"
# Register a test user
EMAIL="scan_$(uuidgen 2>/dev/null || date +%s)@test.com"
PASSWD="Password123!"
ORG="ScanOrg_$(date +%s)"
REG_DATA="{\"email\":\"$EMAIL\",\"username\":\"scanuser\",\"password\":\"$PASSWD\",\"organization_name\":\"$ORG\"}"
expect "POST" "/auth/register" "201" "$REG_DATA" "POST /auth/register (new user)"

expect "POST" "/auth/login" "200" "{\"identifier\":\"$EMAIL\",\"password\":\"$PASSWD\"}" "POST /auth/login"
expect "GET" "/auth/me" "200" "" "GET /auth/me (profile)"
expect "POST" "/auth/refresh" "200" "" "POST /auth/refresh"

echo ""
echo "--- Time Entries ---"
expect "GET" "/time-entries" "200" "" "GET /time-entries (list)"
expect_warn "POST" "/time-entries" "400" "{}" "POST /time-entries (empty body — expect 400)"

echo ""
echo "--- Projects ---"
expect "POST" "/projects" "400" "{}" "POST /projects (empty body — expect 400)"
expect "GET" "/projects" "200" "" "GET /projects (list)"

echo ""
echo "--- Contracts ---"
expect "POST" "/contracts" "400" "{}" "POST /contracts (empty body — expect 400)"
expect "GET" "/contracts" "200" "" "GET /contracts (list)"

echo ""
echo "--- Customers ---"
expect "POST" "/customers" "400" "{}" "POST /customers (empty body — expect 400)"
expect "GET" "/customers" "200" "" "GET /customers (list)"

echo ""
echo "--- Org Hierarchy: Units ---"
expect "GET" "/units" "200" "" "GET /units (list)"
expect "GET" "/units/tree" "200" "" "GET /units/tree"

echo ""
echo "--- Adversarial / Edge Cases ---"
expect "GET" "/auth/me" "401" "" "GET /auth/me (no cookie — expect 401)"
expect "POST" "/auth/register" "400" "{}" "POST /auth/register (empty body — expect 400)"
expect "POST" "/time-entries" "400" "{\"hours\":-1}" "POST /time-entries (negative hours — expect 400)"
expect "GET" "/units/00000000-0000-0000-0000-000000000000" "404" "" "GET /units (non-existent — expect 404)"

echo ""
echo "============================================"
echo "  SCAN RESULTS"
echo "  PASS: $PASS  |  FAIL: $FAIL  |  WARN: $WARN"
echo "============================================"
if [ "$FAIL" -gt 0 ]; then
  echo "  FAILURES:"
  echo -e "$ERRORS"
  exit 1
else
  echo "  All probes completed successfully."
fi
