package alerts

import "testing"

// Red-first regression tests for T-21b decision 46 (the 7/21 duplicate-Cliq
// incident): outbound alert delivery must be OPT-IN. A dev server started from
// backend/.env holds live Zoho/SMTP creds; before this gate it became a second
// live sender next to prod. Only an explicit ALERTS_ENABLED=1 (set by
// docker-compose.prod.yml alone) may enable delivery.

func TestLoadConfigDeliveryDisabledByDefault(t *testing.T) {
	// No ALERTS_ENABLED in the environment — the dev-server case.
	t.Setenv("ALERTS_ENABLED", "")
	if LoadConfig().Enabled {
		t.Fatal("alert delivery must be disabled when ALERTS_ENABLED is unset")
	}
}

func TestLoadConfigDeliveryEnabledExplicitly(t *testing.T) {
	t.Setenv("ALERTS_ENABLED", "1")
	if !LoadConfig().Enabled {
		t.Fatal("ALERTS_ENABLED=1 must enable alert delivery")
	}
}

func TestLoadConfigDeliveryValueIsStrict(t *testing.T) {
	// Anything but the literal "1" stays disabled — a typo must fail closed
	// (silent in dev), never open (double-sending to the live channel).
	for _, v := range []string{"true", "0", "yes", "TRUE"} {
		t.Setenv("ALERTS_ENABLED", v)
		if LoadConfig().Enabled {
			t.Fatalf("ALERTS_ENABLED=%q must NOT enable alert delivery", v)
		}
	}
}
