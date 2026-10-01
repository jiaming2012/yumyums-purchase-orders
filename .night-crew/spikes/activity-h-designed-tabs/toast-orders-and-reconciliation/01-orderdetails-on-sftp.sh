#!/usr/bin/env bash
# 01-orderdetails-on-sftp.sh — spike: decision 188's premise. The Toast SFTP
# export directory HQ already syncs per business date (/<EXPORT_ID>/<YYYYMMDD>/)
# ALSO carries OrderDetails.csv (the order-level file reconciliation joins on).
# READ-ONLY: ReadDir of the export root and of the most recent date dirs,
# dialled EXACTLY the way backend/internal/toast does — golang.org/x/crypto/ssh
# public-key auth with a nil HostKeyCallback (AWS Transfer publishes no stable
# host key) + github.com/pkg/sftp — in a throwaway Go module under TMPDIR.
# (CORRECTION, run 1 → could-not-run: the OpenSSH `sftp` CLI's handshake was
# closed by AWS Transfer after KEX; the production libraries are the honest
# dial path.) Credentials come from backend/.env (TOAST_SFTP_KEY_PATH, relative
# to backend/, plus _USER/_HOST/TOAST_EXPORT_ID) and are never printed. The
# finding is the FILE SET per date dir (B-216), not a sample.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  OrderDetails.csv present in every listed recent date dir.
#   exit 1  connected, listed, and the file is ABSENT → decision 188 falsified;
#           H3 lands on the fixture loader and smtp-toast-ingest returns to PLANNED.
#   exit 2  could not connect / no credentials.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
command -v go >/dev/null || cannot_run "go not on PATH"
ENVF="$REPO_ROOT/backend/.env"; [ -f "$ENVF" ] || cannot_run "backend/.env not found"
getv() { sed -n "s/^$1=//p" "$ENVF" | head -1 | tr -d '"' | tr -d "'"; }
KEY="$(getv TOAST_SFTP_KEY_PATH)"; USER_="$(getv TOAST_SFTP_USER)"; HOST="$(getv TOAST_SFTP_HOST)"; EXP="$(getv TOAST_EXPORT_ID)"
[ -n "$USER_" ] || USER_="YumYumsExportUser"; [ -n "$EXP" ] || EXP="113866"
[ -n "$HOST" ] || HOST="s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22"
KEY="${KEY/#\~/$HOME}"
if [ -n "$KEY" ] && [ ! -f "$KEY" ]; then for base in "$REPO_ROOT/backend" "$REPO_ROOT"; do [ -f "$base/$KEY" ] && KEY="$base/$KEY" && break; done; fi
[ -n "$KEY" ] && [ -f "$KEY" ] || cannot_run "TOAST_SFTP_KEY_PATH unset or file missing after resolving against backend/ and the repo root (value not printed)"
case "$HOST" in *:*) ;; *) HOST="$HOST:22";; esac
echo "# target: sftp://$USER_@$HOST/$EXP/ (READ-ONLY ReadDir via the production libraries; key path not printed)"
W="${TMPDIR:-/tmp}/spike-h3-sftp-$$"; mkdir -p "$W"; trap 'rm -rf "$W"' EXIT
cat > "$W/main.go" <<'GO'
package main
import ("fmt";"net";"os";"sort";"strings";"time";"github.com/pkg/sftp";"golang.org/x/crypto/ssh")
func main(){
  key, err := os.ReadFile(os.Args[1]); if err != nil { fmt.Println("CANNOT read key:", err); os.Exit(2) }
  signer, err := ssh.ParsePrivateKey(key); if err != nil { fmt.Println("CANNOT parse key:", err); os.Exit(2) }
  cfg := &ssh.ClientConfig{User: os.Args[2], Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
    HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error { return nil }, Timeout: 30 * time.Second}
  conn, err := ssh.Dial("tcp", os.Args[3], cfg); if err != nil { fmt.Println("CANNOT dial:", err); os.Exit(2) }
  defer conn.Close()
  c, err := sftp.NewClient(conn); if err != nil { fmt.Println("CANNOT sftp:", err); os.Exit(2) }
  defer c.Close()
  root := "/" + os.Args[4]
  ents, err := c.ReadDir(root); if err != nil { fmt.Println("CANNOT ReadDir", root, ":", err); os.Exit(2) }
  var dates []string
  for _, e := range ents { if e.IsDir() && len(e.Name()) == 8 { dates = append(dates, e.Name()) } }
  sort.Strings(dates)
  if len(dates) == 0 { fmt.Println("CANNOT: no YYYYMMDD dirs under", root); os.Exit(2) }
  if len(dates) > 3 { dates = dates[len(dates)-3:] }
  fmt.Println("# most recent date dirs:", strings.Join(dates, " "))
  missing := 0
  for _, d := range dates {
    fe, err := c.ReadDir(root + "/" + d); if err != nil { fmt.Println("CANNOT ReadDir", d, ":", err); os.Exit(2) }
    var names []string; has := false
    for _, f := range fe { names = append(names, f.Name()); if f.Name() == "OrderDetails.csv" { has = true } }
    sort.Strings(names)
    fmt.Printf("# %s/%s: %s\n", root, d, strings.Join(names, " "))
    if !has { fmt.Println("#   ↑ OrderDetails.csv ABSENT"); missing++ }
  }
  if missing > 0 { fmt.Println("RED: OrderDetails.csv absent in", missing, "of", len(dates), "date dirs"); os.Exit(1) }
  fmt.Println("OK")
}
GO
( cd "$W" && go mod init spikesftp >/dev/null 2>&1 && GOFLAGS=-mod=mod go get github.com/pkg/sftp@latest golang.org/x/crypto/ssh@latest >/dev/null 2>&1 ) || cannot_run "go get pkg/sftp + x/crypto failed (network?)"
set +e; OUT="$(cd "$W" && GOFLAGS=-mod=mod go run . "$KEY" "$USER_" "$HOST" "$EXP" 2>&1)"; RC=$?; set -e
printf '%s\n' "$OUT"
case $RC in 0) ;; 1) fail "OrderDetails.csv absent — decision 188 does not hold; fixture loader + revive smtp-toast-ingest";; *) cannot_run "dial/list failed: $(printf %s "$OUT" | tail -1)";; esac
echo "✅ GREEN — OrderDetails.csv is on the export for every recent date dir listed; H3 reads it off the existing SFTP sync"
