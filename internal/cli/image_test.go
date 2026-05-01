package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteImageScrubRemovesCommonSecretStores(t *testing.T) {
	script := remoteImageScrub()
	for _, want := range []string{
		"/root/.aws",
		"/home/*/.aws",
		"/root/.docker",
		"/.crabbox/actions/*.env.sh",
		"cloud-init clean --logs",
		"journalctl --vacuum-time=1s",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("scrub script missing %q:\n%s", want, script)
		}
	}
}

func TestImagePromoteWritesAWSAMIToConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("CRABBOX_CONFIG", path)
	if err := os.WriteFile(path, []byte("provider: aws\naws:\n  region: eu-west-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	app := App{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	if err := app.imagePromote(t.Context(), []string{"ami-0123456789abcdef0"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := readFileConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AWS == nil || cfg.AWS.AMI != "ami-0123456789abcdef0" {
		t.Fatalf("aws ami not written: %#v", cfg.AWS)
	}
	if !strings.Contains(stdout.String(), "aws.ami=ami-0123456789abcdef0") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}
