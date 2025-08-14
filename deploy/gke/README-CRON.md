# Scheduled metric refresh (GKE CronJobs)

This folder contains two CronJobs for the **ingestor** service:

- `cron-benchmarks-weekly.yaml`: Refresh **benchmarks** weekly (Hugging Face Open LLM Leaderboard, LMArena CSV mirror, HELM JSON, **and** HELM stats.json from public GCS if configured).
- `cron-prices-monthly.yaml`: Refresh **prices** monthly from Artificial Analysis (requires `AA_API_KEY`).

## Apply

```bash
kubectl apply -f deploy/gke/cron-benchmarks-weekly.yaml -n llm-router
kubectl apply -f deploy/gke/cron-prices-monthly.yaml   -n llm-router
```

## Environment variables

These env vars are used by the **ingestor** Pod (set them in your ingestor Deployment or via a ConfigMap/Secret):

**General**  
- `ROUTER_API` — default `http://api:8080/ingest`

**HELM generic JSON (optional)**  
- `HELM_JSON_URLS` — comma-separated list of JSON export URLs you host

**HELM via public GCS (optional)**  
- `HELM_GCS_STATS_URLS` — direct HTTPS URLs to `stats.json` files (e.g., `https://storage.googleapis.com/crfm-helm-public/lite/benchmark_output/runs/v1.0.0/<RUN_ID>/stats.json`)  
  _or_  
- `HELM_GCS_HTTP_PREFIX` — e.g., `https://storage.googleapis.com/crfm-helm-public/lite/benchmark_output`  
- `HELM_SUITE_VERSION` — e.g., `v1.0.0`  
- `HELM_RUN_IDS` — comma-separated run IDs to build the stats.json URLs

**Artificial Analysis (optional, for prices & extra metrics)**  
- `AA_API_KEY` — **store as a Kubernetes Secret**, not a ConfigMap.

### Where to put envs

In your `deploy/gke/deployment-ingestor.yaml`, add:
```yaml
env:
  - name: ROUTER_API
    value: "http://api:8080/ingest"
  - name: SOURCES
    value: "open-llm-leaderboard,lmarena,helm,helm_gcs"
  - name: HELM_GCS_HTTP_PREFIX
    value: "https://storage.googleapis.com/crfm-helm-public/lite/benchmark_output"
  - name: HELM_SUITE_VERSION
    value: "v1.0.0"
  - name: HELM_RUN_IDS
    value: "openai-gpt-4o_2024-07-01,anthropic-claude-3.5-sonnet_2024-07-01"
  - name: HELM_JSON_URLS
    value: "https://your-bucket/helm-custom.json"
  - name: AA_API_KEY
    valueFrom:
      secretKeyRef:
        name: router-secrets
        key: AA_API_KEY
```

For quick reference, these are also documented in the main README.
