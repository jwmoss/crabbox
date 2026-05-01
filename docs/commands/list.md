# list

`crabbox list` shows current Crabbox machines.

```sh
crabbox list
crabbox list --provider aws
crabbox list --provider blacksmith-testbox
crabbox list --json
```

`crabbox pool list` remains as a compatibility alias.

In `blacksmith-testbox` mode this forwards to `blacksmith testbox list`. JSON output is not supported because the Blacksmith CLI owns the list formatting.

Flags:

```text
--provider hetzner|aws|blacksmith-testbox
--json
```
