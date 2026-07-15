# payments-api-demo

Go demo app (api + worker) used to exercise the ZTA pipeline end-to-end and the
GUAC blast-radius on a non-Python ecosystem. Paired with `analytics-engine-demo`.

## Components

- `services/api/` — stdlib HTTP service (`/health`, `/charge`, `/quote`), distroless static.
  `/quote` validates `amount>0` and `currency ∈ {USD,EUR,RON}` and computes `fee`/`net`
  (pure function `computeQuote`, unit-tested).
- `services/worker/` — HTTP-less process loop, distroless static; the pure function
  `parseInterval()` is extracted to be testable.
- `.github/workflows/` — **modular** pipeline (orchestrator `ci-cd.yaml` + `job-*.yml`),
  with the per-service jobs **parameterized** (api/worker) via a `service` input.
- `security-policy.yaml` — input for `policyAttestor-action`.
- `vex.json` — empty OpenVEX (placeholder).

Tests use the **stdlib** `testing`/`httptest` — zero new dependencies, and the
`_test.go` files never enter the binary/image (the distroless SBOM stays clean).

**Images produced (after push):**

- `ghcr.io/sabinghost19/payments-api@sha256:...`
- `ghcr.io/sabinghost19/payments-worker@sha256:...`

**Expected cluster state:** both ZTAs `Verified` (minimal SBOM, 0 Trivy CVEs → the
strict SCA admits them).

## Modular pipeline (api + worker parameterized)

`ci-cd.yaml` is a thin orchestrator that calls reusable workflows (`job-*.yml`).
The per-service jobs (`build-push`, `scan-image`, `attestations`, `sign`) are
called **twice** (api/worker) via a `service` input.

Order: `build-metadata` (go build) + `unit-tests` (`go test ./...`, **gates** the
build) + `security-scan` (gitleaks/Semgrep/checkov on source — gitleaks **BLOCKING**,
gates the build) → `build-push` → `scan-image` (Trivy `trivy-action@v0.36.0`) →
`attestations` (SBOM/OpenVEX/VBBI/ZTA-policy + **`security-scan/v1`** via
`SabinGhost19/security-scan-attestorAction@v1.0.1`) + `slsa-provenance` →
`sign` → `bump-manifests`. `security-scan` runs once over the whole repo (a secret
anywhere blocks both images); the `security-scan/v1` attestation is signed per-image
in `job-attestations.yml`. The first step of every job is `harden-runner`
(audit, pinned by commit-SHA); `unit-tests`/`build-metadata` use `actions/setup-go@v5` (Go 1.26).

**Keyless identities** (add to `trustedIssuers`):

```text
https://github.com/SabinGhost19/payments-api-demo/.github/workflows/job-attestations.yml@refs/heads/main
https://github.com/SabinGhost19/payments-api-demo/.github/workflows/job-sign.yml@refs/heads/main
```

## Local tests

```bash
( cd services/api && go test ./... )
( cd services/worker && go test ./... )
```

## First-run setup

The pipeline reuses the manifests repo `SabinGhost19/vulfastapi-manifests-samples`
(the same one `demo-app` and `analytics` use). It is mirrored locally as
`manifests-demo-app/` in this workspace; the sub-path is `payments-api/`.

1. Create the source repo on GitHub:

   ```bash
   gh repo create SabinGhost19/payments-api-demo --public --source=. --remote=origin
   ```

2. Add secrets (reusable with the `vulfastapi` ones):

   ```bash
   gh secret set VBBI_HMAC_KEY --body "<same-key-as-vulfastapi>"
   gh secret set MANIFESTS_REPO_TOKEN --body "<PAT with repo scope on vulfastapi-manifests-samples>"
   ```

3. Make sure the manifests exist in the shared repo (sub-path `payments-api/`):

   ```bash
   git clone https://github.com/SabinGhost19/vulfastapi-manifests-samples
   cd vulfastapi-manifests-samples
   mkdir -p payments-api/api payments-api/worker
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/sca.yaml ./payments-api/
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/api/zta-api.yaml ./payments-api/api/
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/worker/zta-worker.yaml ./payments-api/worker/
   git add -A && git commit -m "add payments-api manifests" && git push
   ```

4. Push the source repo:

   ```bash
   cd ../payments-api-demo
   git init -b main && git add -A && git commit -m "init"
   git remote add origin https://github.com/SabinGhost19/payments-api-demo.git
   git push -u origin main
   ```

5. The pipeline runs automatically. Inspect the OCI artifacts:

   ```bash
   cosign tree ghcr.io/sabinghost19/payments-api:sha-<commit>
   cosign tree ghcr.io/sabinghost19/payments-worker:sha-<commit>
   ```

6. Apply in the cluster:

   ```bash
   kubectl apply -f payments-api/sca.yaml
   kubectl apply -f payments-api/api/zta-api.yaml
   kubectl apply -f payments-api/worker/zta-worker.yaml
   ```

## QA — Live Test Report (2026-06-24)

TC-01..03 verified on the live k3s cluster; TC-04 in the CI pipeline. Full QA test cases + timing in
[`DOC/qa/20-payments.md`](../../DOC/qa/20-payments.md). **Result: 4/4 PASS (2 negative).**

| TC | What is tested | Expected behaviour | Status |
|---|---|---|---|
| PAY-01 | clean image vs strict policy | `payments-api` → Verified/Running/**Compliant** (only app to pass the Medium/Kill gate) | PASS |
| PAY-02 | runtime enforcement provisioned | `runtimeEnforcement.installed=true, talonRulePatched=true`; `zta-default-payments-api-isolate` Talon rule | PASS |
| PAY-03 | **NEGATIVE** — egress drift | `payments-worker` manifest egress `kafka` ∉ attested policy → `Failed_SupplyChain` (`egress namespace 'kafka' is not allowed by attested policy`), not deployed | PASS |
| PAY-04 | **NEGATIVE** — gitleaks pre-build secret gate (CI) | a planted fake AWS key (`services/worker/local_secrets.yaml`) → `security-scan` job fails (`gitleaks found 2 secret(s): aws-access-token + generic-api-key`, exit 1) → `build-push-api`/`build-push-worker` skipped, no image built | PASS |

The worker failing is the **correct** outcome — the runtime network policy is bound to the signed
zta-policy attestation; a drifted manifest is rejected. PAY-04 adds the **shift-left** layer: a committed
secret is caught by gitleaks and blocks the build before any image is produced.
