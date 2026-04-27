#!/usr/bin/env bash

set -euo pipefail

API_BASE="${API_BASE:-https://api.cloudflare.com/client/v4}"
API_TOKEN="${API_TOKEN:-}"
ACCOUNT_ID="${ACCOUNT_ID:-}"
ZONE_ID="${ZONE_ID:-}"
HOSTNAME="${HOSTNAME:-}"

CHECK_WRITE=false
CHECK_DNS_WRITE=false

TMP_TUNNEL_ID=""
TMP_DNS_RECORD_ID=""

log() {
  printf '[INFO] %s\n' "$*"
}

pass() {
  printf '[PASS] %s\n' "$*"
}

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Verify Cloudflare token permissions required by cloudflaretunnel-operator.

Usage:
  verify-cloudflare-token-permissions.sh \
    --api-token <token> \
    --account-id <account-id> \
    [--zone-id <zone-id>] \
    [--hostname <hostname-in-zone>] \
    [--write-check] \
    [--dns-write-check] \
    [--api-base <url>]

Checks:
  Default:
    - Tunnel read check (list tunnels)
    - DNS read check (if --zone-id provided)

  --write-check:
    - Tunnel create
    - Tunnel token read
    - Tunnel configuration update
    - Tunnel delete

  --dns-write-check:
    - DNS record create, update, delete
    - Requires --zone-id and --hostname

Examples:
  # Read-only checks
  ./hack/verify-cloudflare-token-permissions.sh \
    --api-token "$API_TOKEN" \
    --account-id "$ACCOUNT_ID" \
    --zone-id "$ZONE_ID"

  # Full coverage checks used by the operator
  ./hack/verify-cloudflare-token-permissions.sh \
    --api-token "$API_TOKEN" \
    --account-id "$ACCOUNT_ID" \
    --zone-id "$ZONE_ID" \
    --hostname "permcheck.example.com" \
    --write-check \
    --dns-write-check
EOF
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

cleanup() {
  if [[ -n "$TMP_DNS_RECORD_ID" && -n "$ZONE_ID" ]]; then
    curl -sS -X DELETE \
      "${API_BASE}/zones/${ZONE_ID}/dns_records/${TMP_DNS_RECORD_ID}" \
      -H "Authorization: Bearer ${API_TOKEN}" \
      -H "Content-Type: application/json" >/dev/null || true
  fi

  if [[ -n "$TMP_TUNNEL_ID" ]]; then
    curl -sS -X DELETE \
      "${API_BASE}/accounts/${ACCOUNT_ID}/cfd_tunnel/${TMP_TUNNEL_ID}" \
      -H "Authorization: Bearer ${API_TOKEN}" \
      -H "Content-Type: application/json" >/dev/null || true
  fi
}

cf_request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"

  local body_file
  body_file="$(mktemp)"
  local status_code

  if [[ -n "$data" ]]; then
    status_code="$(curl -sS -o "$body_file" -w "%{http_code}" -X "$method" \
      "${API_BASE}${path}" \
      -H "Authorization: Bearer ${API_TOKEN}" \
      -H "Content-Type: application/json" \
      --data "$data")"
  else
    status_code="$(curl -sS -o "$body_file" -w "%{http_code}" -X "$method" \
      "${API_BASE}${path}" \
      -H "Authorization: Bearer ${API_TOKEN}" \
      -H "Content-Type: application/json")"
  fi

  if [[ ! "$status_code" =~ ^2 ]]; then
    printf 'HTTP %s for %s %s\n' "$status_code" "$method" "$path" >&2
    cat "$body_file" >&2
    rm -f "$body_file"
    return 1
  fi

  if ! jq -e '.success == true' "$body_file" >/dev/null 2>&1; then
    printf 'API failure for %s %s\n' "$method" "$path" >&2
    cat "$body_file" >&2
    rm -f "$body_file"
    return 1
  fi

  cat "$body_file"
  rm -f "$body_file"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --api-token)
      API_TOKEN="${2:-}"
      shift 2
      ;;
    --account-id)
      ACCOUNT_ID="${2:-}"
      shift 2
      ;;
    --zone-id)
      ZONE_ID="${2:-}"
      shift 2
      ;;
    --hostname)
      HOSTNAME="${2:-}"
      shift 2
      ;;
    --write-check)
      CHECK_WRITE=true
      shift
      ;;
    --dns-write-check)
      CHECK_DNS_WRITE=true
      shift
      ;;
    --api-base)
      API_BASE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1 (use --help)"
      ;;
  esac
done

require_cmd curl
require_cmd jq
require_cmd openssl

[[ -n "$API_TOKEN" ]] || fail "--api-token is required"
[[ -n "$ACCOUNT_ID" ]] || fail "--account-id is required"

if [[ "$CHECK_DNS_WRITE" == true ]]; then
  [[ -n "$ZONE_ID" ]] || fail "--zone-id is required for --dns-write-check"
  [[ -n "$HOSTNAME" ]] || fail "--hostname is required for --dns-write-check"
fi

trap cleanup EXIT

log "Checking tunnel read permission (list tunnels)"
cf_request GET "/accounts/${ACCOUNT_ID}/cfd_tunnel?is_deleted=false&per_page=1" >/dev/null
pass "tunnel read check passed"

if [[ -n "$ZONE_ID" ]]; then
  log "Checking DNS read permission (list records)"
  cf_request GET "/zones/${ZONE_ID}/dns_records?per_page=1" >/dev/null
  pass "DNS read check passed"
else
  log "Skipping DNS read check: no --zone-id provided"
fi

if [[ "$CHECK_WRITE" == true ]]; then
  local_tunnel_name="permcheck-$(date +%s)-$RANDOM"
  local_secret="$(openssl rand -base64 32 | tr -d '\n')"

  log "Checking tunnel write permission (create tunnel)"
  create_resp="$(cf_request POST "/accounts/${ACCOUNT_ID}/cfd_tunnel" "{\"name\":\"${local_tunnel_name}\",\"secret\":\"${local_secret}\"}")"
  TMP_TUNNEL_ID="$(printf '%s' "$create_resp" | jq -r '.result.id')"
  [[ -n "$TMP_TUNNEL_ID" && "$TMP_TUNNEL_ID" != "null" ]] || fail "failed to parse created tunnel id"
  pass "tunnel create check passed"

  log "Checking tunnel token read permission"
  cf_request GET "/accounts/${ACCOUNT_ID}/cfd_tunnel/${TMP_TUNNEL_ID}/token" >/dev/null
  pass "tunnel token read check passed"

  log "Checking tunnel configuration update permission"
  cf_request PUT "/accounts/${ACCOUNT_ID}/cfd_tunnel/${TMP_TUNNEL_ID}/configurations" \
    '{"config":{"ingress":[{"service":"http_status:404"}]}}' >/dev/null
  pass "tunnel configuration update check passed"

  log "Cleaning up temporary tunnel"
  cf_request DELETE "/accounts/${ACCOUNT_ID}/cfd_tunnel/${TMP_TUNNEL_ID}" >/dev/null
  TMP_TUNNEL_ID=""
  pass "tunnel delete check passed"
fi

if [[ "$CHECK_DNS_WRITE" == true ]]; then
  dns_name="${HOSTNAME}"
  dns_target="permcheck-target.example.com"

  log "Checking DNS write permission (create CNAME)"
  dns_create_resp="$(cf_request POST "/zones/${ZONE_ID}/dns_records" \
    "{\"type\":\"CNAME\",\"name\":\"${dns_name}\",\"content\":\"${dns_target}\",\"ttl\":1,\"proxied\":true}")"
  TMP_DNS_RECORD_ID="$(printf '%s' "$dns_create_resp" | jq -r '.result.id')"
  [[ -n "$TMP_DNS_RECORD_ID" && "$TMP_DNS_RECORD_ID" != "null" ]] || fail "failed to parse created DNS record id"
  pass "DNS create check passed"

  log "Checking DNS write permission (update CNAME)"
  cf_request PUT "/zones/${ZONE_ID}/dns_records/${TMP_DNS_RECORD_ID}" \
    "{\"type\":\"CNAME\",\"name\":\"${dns_name}\",\"content\":\"${dns_target}\",\"ttl\":1,\"proxied\":true}" >/dev/null
  pass "DNS update check passed"

  log "Cleaning up temporary DNS record"
  cf_request DELETE "/zones/${ZONE_ID}/dns_records/${TMP_DNS_RECORD_ID}" >/dev/null
  TMP_DNS_RECORD_ID=""
  pass "DNS delete check passed"
fi

pass "all requested permission checks passed"
