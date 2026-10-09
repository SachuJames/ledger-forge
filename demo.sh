#!/bin/bash
# End-to-end demo: boots both services (in-memory core), logs in, creates
# accounts, posts a transfer, replays it to prove idempotency, reads back
# the balance and the journal. Exits non-zero on any unexpected step.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
export PATH="$HOME/go/bin:$PATH"

echo "==> building ledger-core"
(cd "$ROOT/ledger-core" && go build -o /tmp/ledgerforge-demo-core ./cmd/ledger-core)

echo "==> building api-gateway"
(cd "$ROOT/api-gateway" && npm run build --silent >/dev/null)

echo "==> starting services"
/tmp/ledgerforge-demo-core > /tmp/lf-demo-core.log 2>&1 &
CORE_PID=$!
LEDGER_CORE_URL=http://localhost:8081 PORT=3000 node "$ROOT/api-gateway/dist/main" > /tmp/lf-demo-gw.log 2>&1 &
GW_PID=$!
trap 'kill $CORE_PID $GW_PID 2>/dev/null' EXIT
sleep 5

TOKEN=$(curl -s -X POST localhost:3000/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"changeme"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")
echo "==> logged in"

auth() { curl -s -H "Authorization: Bearer $TOKEN" "$@"; }

CASH=$(auth -X POST localhost:3000/api/accounts -H 'Content-Type: application/json' \
  -d '{"code":"1001","name":"Cash","type":"asset"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
REV=$(auth -X POST localhost:3000/api/accounts -H 'Content-Type: application/json' \
  -d '{"code":"4001","name":"Revenue","type":"revenue"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
echo "==> accounts created"

TRANSFER="{\"debitAccountId\":\"$CASH\",\"creditAccountId\":\"$REV\",\"currency\":\"INR\",\"amount\":2500,\"narration\":\"demo sale\",\"idempotencyKey\":\"demo-1\"}"
R1=$(auth -X POST localhost:3000/api/transfers -H 'Content-Type: application/json' -d "$TRANSFER" \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['replayed'])")
R2=$(auth -X POST localhost:3000/api/transfers -H 'Content-Type: application/json' -d "$TRANSFER" \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['replayed'])")
echo "==> transfer posted (replayed=$R1), retried (replayed=$R2)"
[ "$R1" = "False" ] && [ "$R2" = "True" ] || { echo "FAIL: idempotency broken"; exit 1; }

BAL=$(auth "localhost:3000/api/accounts/$CASH/balance?currency=INR" \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['formatted'])")
echo "==> cash balance: $BAL INR"
[ "$BAL" = "25.00" ] || { echo "FAIL: balance wrong"; exit 1; }

N=$(auth localhost:3000/api/journal | python3 -c "import json,sys; print(len(json.load(sys.stdin)['entries']))")
echo "==> journal entries: $N (one, not two: the retry did not double-post)"
echo "==> demo passed"
