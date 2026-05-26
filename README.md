# payments-api-demo

Demo aplicație Go (api + worker) folosită pentru testarea pipeline-ului ZTA
end-to-end și GUAC blast-radius pe ecosistem distinct de Python. Pereche cu
`analytics-engine-demo`.

## Componente

- `services/api/` — HTTP service (`/health`, `/charge`), distroless static.
- `services/worker/` — process loop fără HTTP, distroless static.
- `.github/workflows/ci-cd.yaml` — pipeline complet (build + push + atestări
  + SLSA v1.0 + GitOps bump). Versiune adaptată din `vulfastapi/ci-cd.yaml`
  cu job-uri separate per microserviciu.
- `security-policy.yaml` — input pentru `policyAttestor-action`.
- `vex.json` — OpenVEX gol (placeholder).

**Imagini produse (după push):**

- `ghcr.io/sabinghost19/payments-api@sha256:...`
- `ghcr.io/sabinghost19/payments-worker@sha256:...`

**Stare așteptată în cluster:** ambele ZTA-uri `Verified` (SBOM minimal,
0 CVE-uri Trivy → strict SCA acceptă).

## Setup primă rulare

Pipeline-ul reutilizează repo-ul de manifeste existent
`SabinGhost19/vulfastapi-manifests-samples` (același pe care îl folosește
`demo-app`). Locally este oglindit ca `demo-app-manifests-samples/` în
acest workspace.

1. Creează repo-ul source pe GitHub:

   ```bash
   gh repo create SabinGhost19/payments-api-demo --public --source=. --remote=origin
   ```

2. Adaugă secret-uri (reutilizabile cu cele din `vulfastapi`):

   ```bash
   gh secret set VBBI_HMAC_KEY --body "<same-key-as-vulfastapi>"
   gh secret set MANIFESTS_REPO_TOKEN --body "<PAT cu repo scope pe vulfastapi-manifests-samples>"
   ```

3. Asigură-te că manifestele există în repo-ul shared. Local sunt deja la
   `demo-app-manifests-samples/payments-api/`, dar trebuie să le pui și în
   `SabinGhost19/vulfastapi-manifests-samples` la sub-path-ul `payments-api/`:

   ```bash
   git clone https://github.com/SabinGhost19/vulfastapi-manifests-samples
   cd vulfastapi-manifests-samples
   mkdir -p payments-api/api payments-api/worker
   cp ../customCRD/demo-app-manifests-samples/payments-api/sca.yaml ./payments-api/
   cp ../customCRD/demo-app-manifests-samples/payments-api/api/zta-api.yaml ./payments-api/api/
   cp ../customCRD/demo-app-manifests-samples/payments-api/worker/zta-worker.yaml ./payments-api/worker/
   git add -A && git commit -m "add payments-api manifests" && git push
   ```

4. Push source repo:

   ```bash
   cd ../payments-api-demo
   git init -b main
   git add -A && git commit -m "init"
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
