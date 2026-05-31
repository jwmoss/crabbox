package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPrewarmDryRunPlansHydratedLease(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("CRABBOX_CONFIG", filepath.Join(dir, ".crabbox.yaml"))
	if err := os.WriteFile(filepath.Join(dir, ".crabbox.yaml"), []byte(`provider: azure
target: linux
class: standard
actions:
  workflow: hydrate.yml
  job: hydrate
  ref: main
cache:
  volumes:
    - name: pnpm
      key: repo-pnpm
      path: /var/cache/crabbox/pnpm
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	app := App{Stdout: &stdout, Stderr: &stderr}
	if err := app.Run(context.Background(), []string{"prewarm", "--dry-run", "--provider", "azure", "--azure-backend", "vm", "--desktop", "--browser", "--os", "ubuntu:24.04", "--probe-command", "node -v && pnpm -v"}); err != nil {
		t.Fatalf("prewarm dry-run failed: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"crabbox warmup --provider azure --azure-backend vm --desktop --browser --os ubuntu:24.04 --keep=true",
		"crabbox actions hydrate --azure-backend vm --provider azure --target linux",
		"--workflow hydrate.yml --job hydrate --ref main",
		"crabbox run --azure-backend vm --provider azure --target linux",
		"--no-sync --no-hydrate --shell -- 'node -v && pnpm -v'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--cache-volume") {
		t.Fatalf("azure prewarm should not request unsupported cache volume flags:\n%s", got)
	}
}

func TestPrewarmDryRunKeepsBlacksmithProviderOwned(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("CRABBOX_CONFIG", filepath.Join(dir, ".crabbox.yaml"))
	if err := os.WriteFile(filepath.Join(dir, ".crabbox.yaml"), []byte(`provider: blacksmith-testbox
blacksmith:
  org: example-org
  workflow: testbox.yml
  job: check
cache:
  volumes:
    - name: pnpm
      key: repo-pnpm
      path: /var/cache/crabbox/pnpm
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	app := App{Stdout: &stdout, Stderr: &stderr}
	if err := app.Run(context.Background(), []string{"prewarm", "--dry-run", "--provider", "blacksmith-testbox", "--blacksmith-workflow", "testbox.yml", "--blacksmith-job", "check", "--cache-volume", "pnpm=repo-pnpm:/var/cache/crabbox/pnpm", "--probe-command", "node -v"}); err != nil {
		t.Fatalf("prewarm dry-run failed: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "crabbox warmup --provider blacksmith-testbox") ||
		!strings.Contains(got, "--blacksmith-workflow testbox.yml") ||
		!strings.Contains(got, "--blacksmith-job check") ||
		!strings.Contains(got, "--cache-volume pnpm=repo-pnpm:/var/cache/crabbox/pnpm") {
		t.Fatalf("blacksmith warmup plan missing sticky cache volume:\n%s", got)
	}
	if strings.Contains(got, "actions hydrate") {
		t.Fatalf("blacksmith prewarm should not run local Actions hydration:\n%s", got)
	}
	if !strings.Contains(got, "crabbox run --blacksmith-workflow testbox.yml --blacksmith-job check --provider blacksmith-testbox") ||
		!strings.Contains(got, "--no-sync --no-hydrate --shell -- 'node -v'") {
		t.Fatalf("blacksmith prewarm should still run explicit probe:\n%s", got)
	}
}

func TestPrewarmDryRunMapsGenericWorkflowFlagsForBlacksmith(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("CRABBOX_CONFIG", filepath.Join(dir, ".crabbox.yaml"))
	if err := os.WriteFile(filepath.Join(dir, ".crabbox.yaml"), []byte(`provider: blacksmith-testbox
blacksmith:
  org: example-org
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	app := App{Stdout: &stdout, Stderr: &stderr}
	if err := app.Run(context.Background(), []string{"prewarm", "--dry-run", "--provider", "blacksmith-testbox", "--workflow", "testbox.yml", "--job", "check", "--ref", "main"}); err != nil {
		t.Fatalf("prewarm dry-run failed: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"--blacksmith-workflow testbox.yml",
		"--blacksmith-job check",
		"--blacksmith-ref main",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("blacksmith warmup plan missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "actions hydrate") || strings.Contains(got, "crabbox run") {
		t.Fatalf("blacksmith prewarm should stay provider-owned:\n%s", got)
	}
}

func TestPrewarmDryRunDoesNotBootstrapPondACL(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("CRABBOX_CONFIG", filepath.Join(dir, ".crabbox.yaml"))
	t.Setenv(pondACLAutoBootstrapEnvVar, "1")
	t.Setenv("TS_API_KEY", "tskey-api-stub")
	t.Setenv("CRABBOX_TAILSCALE_AUTH_KEY", "tskey-auth-test")
	if err := os.WriteFile(filepath.Join(dir, ".crabbox.yaml"), []byte(`provider: hetzner
target: linux
tailscale:
  enabled: true
  tags:
    - tag:crabbox
actions:
  workflow: hydrate.yml
  job: hydrate
`), 0o600); err != nil {
		t.Fatal(err)
	}
	stub := &stubPondTailnetACLClient{policy: pondPolicyFixture(pondTailscaleTag(localCoordinatorOwner(), "alpha")), etag: `"v1"`}
	prev := pondTailnetACLClientFactory
	t.Cleanup(func() { pondTailnetACLClientFactory = prev })
	pondTailnetACLClientFactory = func(_ string) pondTailnetACLClient { return stub }

	var stdout, stderr bytes.Buffer
	app := App{Stdout: &stdout, Stderr: &stderr}
	if err := app.Run(context.Background(), []string{"prewarm", "--dry-run", "--provider", "hetzner", "--pond", "alpha"}); err != nil {
		t.Fatalf("prewarm dry-run failed: %v\nstderr=%s", err, stderr.String())
	}
	if atomic.LoadInt32(&stub.gets) != 0 || atomic.LoadInt32(&stub.puts) != 0 {
		t.Fatalf("dry-run touched pond ACL API: gets=%d puts=%d", stub.gets, stub.puts)
	}
}
