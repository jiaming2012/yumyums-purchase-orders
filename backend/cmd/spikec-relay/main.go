// Command spikec-relay — 🛑 SPIKE CODE. NOT PRODUCTION.
//
// Night-crew card C `spike-c-round-trip` (Spike C), run 20260807-2. A ~40-line
// runner whose only job is to give `sync.RunSpikeCRelay` a `main` so a shell
// script can start and stop it. All of the mechanism, and all of the reasoning
// about it, lives in backend/internal/sync/spikec_relay.go — read that file's
// banner first.
//
// It exists at all only because a Go library cannot be executed. It is declared
// as a footprint deviation in the card's merge intent rather than smuggled in:
// the card's footprint named `backend/internal/sync/**`.
//
// 🛑 It is launched by exactly one caller,
// `.night-crew/qa/spike-supabase/spike-c-roundtrip.sh`, and it talks to
// throwaway infrastructure only: the scratch `spike-c-hq` Postgres on a
// Docker-assigned ephemeral port, and spike A's local `spike-supabase` stack.
// NEVER :5433 (that cluster is PRODUCTION and serves hq.yumyums.kitchen),
// never :5434, never a hosted Supabase project.
//
// Every setting is required and there are no defaults. A relay that started
// with a plausible-looking default and pointed at nothing would be the silent
// no-op this whole card exists to retire.
//
//	SPIKE_C_HQ_DSN        postgres://hq:...@127.0.0.1:<ephemeral>/hq_real
//	SPIKE_C_REST_BASE     http://127.0.0.1:<spike A PostgREST port>
//	SPIKE_C_SERVICE_TOKEN HS256 JWT, role=service_role
//	SPIKE_C_SYNC_TABLE    hq_sync_checklists
//
// 🛑 SPIKE_C_APP_SLUG is GONE. Card `app-slug-association` (B-160, run 20260901)
// closed spike B's finding #1: the relay now resolves each projected row's
// app_slug from the template->app association (checklist_templates.app_id,
// migration 0076), per row, instead of taking a constant here. The shell
// harnesses may still export SPIKE_C_APP_SLUG; it is simply no longer read.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	hqsync "github.com/yumyums/hq/internal/sync"
)

func main() {
	cfg := hqsync.SpikeCRelayConfig{
		HQConnStr:    os.Getenv("SPIKE_C_HQ_DSN"),
		RESTBase:     os.Getenv("SPIKE_C_REST_BASE"),
		ServiceToken: os.Getenv("SPIKE_C_SERVICE_TOKEN"),
		SyncTable:    os.Getenv("SPIKE_C_SYNC_TABLE"),
	}

	// The shell blocks on this exact line before it makes the write. Without it
	// the relay races the only NOTIFY the spike ever fires, and a missed
	// notification would red the round trip for a reason that is not the
	// mechanism. Flushed explicitly because the shell reads it through a pipe.
	cfg.Ready = func() {
		fmt.Println("SPIKE_C_RELAY_READY")
		os.Stdout.Sync()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := hqsync.RunSpikeCRelay(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "spikec-relay: %v\n", err)
		os.Exit(1)
	}
}
