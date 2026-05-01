import { describe, expect, it } from "vitest";

import { cloudInit } from "../src/bootstrap";
import type { LeaseConfig } from "../src/config";

const config: LeaseConfig = {
  provider: "aws",
  profile: "project-check",
  class: "standard",
  serverType: "c7a.8xlarge",
  location: "fsn1",
  image: "ubuntu-24.04",
  awsRegion: "eu-west-1",
  awsAMI: "",
  awsSGID: "",
  awsSubnetID: "",
  awsProfile: "",
  awsRootGB: 400,
  capacityMarket: "spot",
  capacityStrategy: "most-available",
  capacityFallback: "on-demand-after-120s",
  capacityRegions: [],
  capacityAvailabilityZones: [],
  sshUser: "crabbox",
  sshPort: "2222",
  providerKey: "crabbox-steipete",
  workRoot: "/work/crabbox",
  ttlSeconds: 1200,
  idleTimeoutSeconds: 360,
  keep: false,
  sshPublicKey: "ssh-ed25519 test",
};

describe("cloud-init bootstrap", () => {
  it("uses retrying package installation in runcmd", () => {
    const got = cloudInit(config);
    expect(got).toContain("package_update: false");
    expect(got).toContain("bash -euxo pipefail <<'BOOT'");
    expect(got).toContain('Acquire::Retries "8";');
    expect(got).toContain("retry apt-get update");
    expect(got).toContain(
      "retry apt-get install -y --no-install-recommends openssh-server ca-certificates curl git rsync jq",
    );
    expect(got).toContain("curl --version >/dev/null");
    expect(got).toContain("test -f /var/lib/crabbox/bootstrapped");
    expect(got).toContain("test -w /work/crabbox");
    expect(got).toContain("touch /var/lib/crabbox/bootstrapped");
    expect(got).not.toContain("\npackages:\n");
    expect(got).not.toContain("go version");
    expect(got).not.toContain("golang-go");
    expect(got).not.toContain("go.dev/dl/go");
    expect(got).not.toContain("/usr/local/go");
    expect(got).not.toContain("node --version");
    expect(got).not.toContain("pnpm --version");
    expect(got).not.toContain("docker --version");
    expect(got).not.toContain("build-essential");
    expect(got).not.toContain("docker.io");
    expect(got).not.toContain("corepack");
  });
});
