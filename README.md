# payments-api-demo

Demo aplicație Go (api + worker) folosită pentru testarea pipeline-ului ZTA
end-to-end și GUAC blast-radius pe ecosistem distinct de Python. Pereche cu
`analytics-engine-demo`.

## Componente

- `services/api/` — HTTP service stdlib (`/health`, `/charge`, `/quote`), distroless static.
  `/quote` validează `amount>0` și `currency ∈ {USD,EUR,RON}` și calculează `fee`/`net`
  (funcția pură `computeQuote`, testată).
- `services/worker/` — process loop fără HTTP, distroless static; funcția pură
  `parseInterval()` este extrasă pentru a fi testabilă.
- `.github/workflows/` — pipeline **modular** (orchestrator `ci-cd.yaml` + `job-*.yml`),
  cu joburile per-serviciu **parametrizate** (api/worker) printr-un input `service`.
- `security-policy.yaml` — input pentru `policyAttestor-action`.
- `vex.json` — OpenVEX gol (placeholder).

Testele sunt scrise cu `testing`/`httptest` din **stdlib** — zero dependențe noi,
iar fișierele `_test.go` nu intră în binar/imagine (SBOM-ul distroless rămâne curat).

**Imagini produse (după push):**

- `ghcr.io/sabinghost19/payments-api@sha256:...`
- `ghcr.io/sabinghost19/payments-worker@sha256:...`

**Stare așteptată în cluster:** ambele ZTA-uri `Verified` (SBOM minimal, 0 CVE-uri
Trivy → strict SCA acceptă).

## Pipeline modular (api + worker parametrizat)

`ci-cd.yaml` este un orchestrator subțire care apelează reusable workflows
(`job-*.yml`). Joburile per-serviciu (`build-push`, `scan-image`, `attestations`,
`sign`) sunt apelate de **două ori** (api/worker) printr-un input `service`.

Ordinea: `build-metadata` (go build) + `unit-tests` (`go test ./...`, **poartă**
build-ul) + `security-scan` (gitleaks/Semgrep/checkov pe sursă — gitleaks **BLOCANT**,
poartă build-ul) → `build-push` → `scan-image` (Trivy `trivy-action@v0.36.0`) →
`attestations` (SBOM/OpenVEX/VBBI/ZTA-policy + **`security-scan/v1`** via
`SabinGhost19/security-scan-attestorAction@v1.0.1`) + `slsa-provenance` →
`sign` → `bump-manifests`. `security-scan` rulează o singură dată pe tot repo-ul
(un secret oriunde blochează ambele imagini); atestarea `security-scan/v1` se semnează
per-imagine în `job-attestations.yml`. Primul pas din fiecare job este `harden-runner`
(audit, fixat pe SHA); `unit-tests`/`build-metadata` folosesc `actions/setup-go@v5` (Go 1.26).

**Identități keyless** (de pus în `trustedIssuers`):

```text
https://github.com/SabinGhost19/payments-api-demo/.github/workflows/job-attestations.yml@refs/heads/main
https://github.com/SabinGhost19/payments-api-demo/.github/workflows/job-sign.yml@refs/heads/main
```

## Teste locale

```bash
( cd services/api && go test ./... )
( cd services/worker && go test ./... )
```

## Setup primă rulare

Pipeline-ul reutilizează repo-ul de manifeste `SabinGhost19/vulfastapi-manifests-samples`
(același pe care îl folosesc `demo-app` și `analytics`). Local este oglindit ca
`manifests-demo-app/` în acest workspace; sub-path-ul este `payments-api/`.

1. Creează repo-ul source pe GitHub:

   ```bash
   gh repo create SabinGhost19/payments-api-demo --public --source=. --remote=origin
   ```

2. Adaugă secrete (reutilizabile cu cele din `vulfastapi`):

   ```bash
   gh secret set VBBI_HMAC_KEY --body "<same-key-as-vulfastapi>"
   gh secret set MANIFESTS_REPO_TOKEN --body "<PAT cu repo scope pe vulfastapi-manifests-samples>"
   ```

3. Asigură-te că manifestele există în repo-ul shared (sub-path `payments-api/`):

   ```bash
   git clone https://github.com/SabinGhost19/vulfastapi-manifests-samples
   cd vulfastapi-manifests-samples
   mkdir -p payments-api/api payments-api/worker
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/sca.yaml ./payments-api/
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/api/zta-api.yaml ./payments-api/api/
   cp ../customCRD/demo-repos-apps/manifests-demo-app/payments-api/worker/zta-worker.yaml ./payments-api/worker/
   git add -A && git commit -m "add payments-api manifests" && git push
   ```

4. Push source repo:

   ```bash
   cd ../payments-api-demo
   git init -b main && git add -A && git commit -m "init"
   git remote add origin https://github.com/SabinGhost19/payments-api-demo.git
   git push -u origin main
   ```

5. Pipeline-ul rulează automat. Verifică artefactele OCI:

   ```bash
   cosign tree ghcr.io/sabinghost19/payments-api:sha-<commit>
   cosign tree ghcr.io/sabinghost19/payments-worker:sha-<commit>
   ```

6. Apply în cluster:

   ```bash
   kubectl apply -f payments-api/sca.yaml
   kubectl apply -f payments-api/api/zta-api.yaml
   kubectl apply -f payments-api/worker/zta-worker.yaml
   ```
