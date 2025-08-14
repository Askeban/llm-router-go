import os, requests, csv, io
from fastapi import FastAPI
from pydantic import BaseModel
from typing import List, Dict, Optional
from datasets import load_dataset
from bs4 import BeautifulSoup
ROUTER_API = os.environ.get("ROUTER_API","http://api:8080/ingest"); AA_API_KEY = os.environ.get("AA_API_KEY")
app = FastAPI(title="Benchmark Ingestor v4")
# put near the top of the function (once), after imports:
ALLOWED_METRIC_NAMES = {
    # classification / QA
    "accuracy", "exact_match", "em", "f1", "macro_f1", "matthews_corrcoef",
    # generation / translation / summarization
    "bleu", "sacrebleu", "meteor", "chrf", "rouge1", "rouge2", "rougel",
    # code
    "pass@1", "pass@k", "humaneval", "human_eval",
    # math / reasoning
    "mmlu", "gsm8k", "math", "bbh",
    # generic score holders
    "score", "auc", "roc_auc", "precision", "recall"
}
ALLOWED_SUBSTRINGS = ["acc", "exact_match", "f1", "bleu", "rouge", "perplex", "auc", "roc", "precision", "recall", "pass@"]

def is_allowed_metric_name(metric_name: str) -> bool:
    n = (metric_name or "").strip().lower()
    if not n:
        return False
    if n in ALLOWED_METRIC_NAMES:
        return True
    return any(s in n for s in ALLOWED_SUBSTRINGS)
class RefreshRequest(BaseModel): sources: Optional[List[str]] = None; models: Optional[List[str]] = None
def push(source: str, rows: List[Dict]): 
    if not rows: return {"ok": True, "count": 0}
    r = requests.post(ROUTER_API, json={"source": source, "metrics": rows}, timeout=60); r.raise_for_status(); return r.json()
def map_metric_to_task_diff(name: str):
    n=name.lower()
    if n=="mmlu": return ("reasoning","medium")
    if n=="bbh": return ("reasoning","hard")
    if n=="gsm8k": return ("math","medium")
    if n=="math": return ("math","hard")
    if n in ("human_eval","humaneval"): return ("coding","medium")
    if "livecodebench" in n: return ("coding","hard")
    if "scicode" in n: return ("coding","hard")
    if n=="ifeval": return ("writing","easy")
    if n=="arena_elo": return ("general", None)
    if "coding" in n and "complex" in n: return ("coding","hard")
    if "coding" in n and "easy" in n: return ("coding","easy")
    if "coding" in n and "medium" in n: return ("coding","medium")
    if "math" in n and "complex" in n: return ("math","hard")
    if "intelligence" in n: return ("reasoning", None)
    return (None, None)
def _coerce_float(v):
    if isinstance(v, (int, float)):
        return float(v)
    if isinstance(v, str):
        try:
            return float(v.strip())
        except Exception:
            return None
    return None

HF_KEYS = ["mmlu","bbh","gsm8k","math","human_eval","humaneval","livecodebench","ifeval","gpqa","scicode"]
HF_MODEL_KEYS = ["model_name","model","name","id","model_id"]

def ingest_artificialanalysis_llms(models: Optional[List[str]]):
    rows=[]; 
    if not AA_API_KEY: return rows
    url="https://artificialanalysis.ai/api/v2/data/llms/models"
    try:
        resp=requests.get(url,headers={"x-api-key":AA_API_KEY},timeout=45)
        if resp.status_code!=200: return rows
        data=resp.json().get("data",[])
        for rec in data:
            name=rec.get("name") or rec.get("id") or rec.get("slug"); 
            if not name: continue
            if models and name not in models: continue
            pricing=rec.get("pricing",{}) or {}
            if "price_1m_input_tokens" in pricing: rows.append({"model_id":name,"name":"cost_in_per_1k","value":float(pricing["price_1m_input_tokens"])/1000.0,"unit":"usd"})
            if "price_1m_output_tokens" in pricing: rows.append({"model_id":name,"name":"cost_out_per_1k","value":float(pricing["price_1m_output_tokens"])/1000.0,"unit":"usd"})
            if "median_output_tokens_per_second" in rec: rows.append({"model_id":name,"name":"tokens_per_second","value":float(rec["median_output_tokens_per_second"]),"unit":"tps"})
            if "median_time_to_first_token_seconds" in rec: rows.append({"model_id":name,"name":"ttft_seconds","value":float(rec["median_time_to_first_token_seconds"]),"unit":"s"})
            evals=rec.get("evaluations",{}) or {}
            for k,v in evals.items():
                if isinstance(v,(int,float)):
                    task,diff = map_metric_to_task_diff(k)
                    item={"model_id":name,"name":k,"value":float(v),"unit":"score"}
                    if task: item["task"]=task
                    if diff: item["difficulty"]=diff
                    rows.append(item)
    except Exception: pass
    return rows
def scrape_artificialanalysis_models_page(models: Optional[List[str]]):
    return []  # disabled to avoid brittle parsing
# ---- REPLACE the whole ingest_open_llm_leaderboard() with this ----
def ingest_open_llm_leaderboard(models: Optional[List[str]]):
    """
    Robust HF fetcher:
      1) discover configs/splits with /splits
      2) page /rows (length<=100) until exhausted (up to ~10k rows default)
      3) extract metric fields into normalized rows
    Datasets we try (in order):
      - open-llm-leaderboard/results
      - open-llm-leaderboard/contents
    """
    import math
    rows = []
    if not hasattr(app.state, "hf_debug"):
        app.state.hf_debug = {}
    dbg = app.state.hf_debug
    headers = {"User-Agent": "llm-router-ingestor/1.0"}

    DATASETS = [
        "open-llm-leaderboard/results",
        "open-llm-leaderboard/contents",
    ]

    def get_splits(ds):
        url = "https://datasets-server.huggingface.co/splits"
        r = requests.get(url, params={"dataset": ds}, headers=headers, timeout=45)
        dbg[f"{ds}_splits_status"] = getattr(r, "status_code", None)
        if r.status_code != 200:
            dbg[f"{ds}_splits_err"] = r.text[:160]
            return []
        j = r.json()
        return j.get("splits", []) or []

    def page_rows(ds, config, split, limit=5000):
        # returns generator of row dicts
        url = "https://datasets-server.huggingface.co/rows"
        offset = 0
        step = 100
        fetched_any = False
        while offset < limit:
            params = {
                "dataset": ds, "config": config, "split": split,
                "offset": str(offset), "length": str(step)
            }
            r = requests.get(url, params=params, headers=headers, timeout=60)
            if r.status_code != 200:
                dbg[f"{ds}_{config}_{split}_rows_err_{offset}"] = r.text[:160]
                break
            j = r.json()
            chunk = j.get("rows") or []
            if not chunk:
                break
            fetched_any = True
            for it in chunk:
                yield it.get("row") or {}
            if len(chunk) < step:
                break
            offset += step
        if not fetched_any:
            dbg[f"{ds}_{config}_{split}_rows_empty"] = True

    def collect_metric_items(rec):
        out = []
        # Some rows flatten metrics as top-level columns, some nest into "metrics"
        metrics_obj = rec.get("metrics") if isinstance(rec.get("metrics"), dict) else {}
        # Candidate model fields
        model = ""
        for mk in HF_MODEL_KEYS:
            mv = rec.get(mk)
            if isinstance(mv, str) and mv.strip():
                model = mv.strip(); break
        if not model:
            # sometimes there's 'repo' or 'hf_repo'
            for mk in ("repo", "hf_repo"):
                mv = rec.get(mk)
                if isinstance(mv, str) and mv.strip():
                    model = mv.strip(); break
        if not model:
            return out
        if models and model not in models:
            return out

        for k in HF_KEYS:
            v = rec.get(k, None)
            if v is None and metrics_obj:
                v = metrics_obj.get(k, None)
            v = _coerce_float(v)
            if v is None:
                continue
            task, diff = map_metric_to_task_diff(k)
            item = {"model_id": model, "name": k, "value": v, "unit": "score"}
            if task: item["task"] = task
            if diff: item["difficulty"] = diff
            out.append(item)
        return out

    for ds in DATASETS:
        try:
            splits = get_splits(ds)
            if not splits:
                continue
            # prefer something like config="default" or split="train", but accept any
            for sp in splits:
                config = sp.get("config"); split = sp.get("split")
                if not (config and split):
                    continue
                cnt_before = len(rows)
                for rec in page_rows(ds, config, split, limit=10000):
                    rows.extend(collect_metric_items(rec))
                if len(rows) > cnt_before:
                    dbg["used"] = f"{ds}:{config}:{split}"
                    return rows
        except Exception as e:
            dbg[f"{ds}_error"] = str(e)
            continue

    # As a tiny last resort, try your older datasets-server helper (kept for compatibility)
    try:
        alt = ingest_open_llm_leaderboard_via_datasets_server(models)
        if alt:
            dbg["used"] = "legacy_datasets_server_fallback"
            return alt
    except Exception as e:
        dbg["legacy_datasets_server_fallback_error"] = str(e)

    return rows
def ingest_open_llm_leaderboard_via_datasets_server(models: Optional[List[str]]):
    """
    Fallback: use the HuggingFace datasets-server API to fetch rows.
    It supports pagination via offset/length. We’ll attempt two dataset names:
      - open-llm-leaderboard/leaderboard
      - open-llm-leaderboard/results
    API docs: https://datasets-server.huggingface.co
    Example:
      https://datasets-server.huggingface.co/rows?dataset=open-llm-leaderboard%2Fleaderboard&config=default&split=train&offset=0&length=1000
    """
    rows = []
    base = "https://datasets-server.huggingface.co/rows"
    candidates = [
        ("open-llm-leaderboard/leaderboard", "default", "train"),
        ("open-llm-leaderboard/results", "default", "train"),
    ]

    def fetch_dataset_rows(dataset, config, split, limit=5000):
        out = []
        offset = 0
        step = 1000
        while offset < limit:
            params = {
                "dataset": dataset,
                "config": config,
                "split": split,
                "offset": str(offset),
                "length": str(step),
            }
            r = requests.get(base, params=params, timeout=60)
            if r.status_code != 200:
                break
            j = r.json()
            rows_chunk = (j.get("rows") or [])
            if not rows_chunk:
                break
            out.extend(rows_chunk)
            if len(rows_chunk) < step:
                break
            offset += step
        return out

    for dataset, config, split in candidates:
        try:
            raw = fetch_dataset_rows(dataset, config, split)
        except Exception:
            continue
        # rows are like {"row": {...}, "row_idx": n}
        for item in raw:
            rec = item.get("row") or {}
            # find model id
            model = ""
            for mk in HF_MODEL_KEYS:
                mv = rec.get(mk)
                if isinstance(mv, str) and mv.strip():
                    model = mv.strip()
                    break
            if not model:
                continue
            if models and model not in models:
                continue

            # metrics: check top-level keys and optional nested 'metrics' dict
            metrics_obj = rec.get("metrics") if isinstance(rec.get("metrics"), dict) else {}
            for k in HF_KEYS:
                v = rec.get(k, None)
                if v is None and metrics_obj:
                    v = metrics_obj.get(k, None)
                v = _coerce_float(v)
                if v is None:
                    continue
                task, diff = map_metric_to_task_diff(k)
                item = {"model_id": model, "name": k, "value": v, "unit": "score"}
                if task: item["task"] = task
                if diff: item["difficulty"] = diff
                rows.append(item)

        if rows:
            break
    return rows
# ---- REPLACE the whole ingest_lmarena() with this ----
def ingest_lmarena(models: Optional[List[str]]):
    """
    Robust LMArena fetcher:
      1) GitHub Releases API for fboulnois/llm-leaderboard-csv (latest)
      2) pick asset that looks like 'lmarena' CSV
      3) parse CSV, extract Elo/score
    """
    import csv, io, re

    if not hasattr(app.state, "lmarena_debug"):
        app.state.lmarena_debug = {}
    dbg = app.state.lmarena_debug
    headers = {"User-Agent": "llm-router-ingestor/1.0"}

    api = "https://api.github.com/repos/fboulnois/llm-leaderboard-csv/releases/latest"
    try:
        r = requests.get(api, headers=headers, timeout=45)
        dbg["releases_status"] = r.status_code
        if r.status_code != 200:
            dbg["releases_err"] = r.text[:200]
            return []
        rel = r.json()
        assets = rel.get("assets") or []
        csv_url = None
        for a in assets:
            name = a.get("name","").lower()
            # seen names: lmarena.csv, lmarena_full.csv, arena.csv, etc.
            if name.endswith(".csv") and "lmarena" in name:
                csv_url = a.get("browser_download_url")
                break
        if not csv_url and assets:
            # fallback: take first CSV if any
            for a in assets:
                if (a.get("name","").lower().endswith(".csv")):
                    csv_url = a.get("browser_download_url"); break
        if not csv_url:
            dbg["no_csv_asset"] = True
            return []

        resp = requests.get(csv_url, headers=headers, timeout=60)
        dbg["csv_status"] = resp.status_code
        if resp.status_code != 200:
            dbg["csv_err"] = resp.text[:200]
            return []

        rows = []
        rd = csv.DictReader(io.StringIO(resp.text))
        for r in rd:
            model = (
                r.get("model") or r.get("Model") or r.get("model_name") or
                r.get("name") or r.get("Model Name") or ""
            ).strip()
            if not model:
                continue
            if models and model not in models:
                continue

            score = None
            for k in ("arena_score","arenaScore","Elo","elo","Score","score","overall"):
                fv = _coerce_float(r.get(k))
                if fv is not None:
                    score = fv; break
            if score is not None:
                rows.append({"model_id": model, "name": "arena_elo", "value": score, "unit": "elo"})
        if rows:
            dbg["used"] = csv_url
        return rows
    except Exception as e:
        dbg["error"] = str(e)
        return []
def ingest_helm_gcs(models: Optional[List[str]]):
    """
    Fetch HELM aggregated stats from the public GCS bucket via the JSON API.

    Env (all optional; sensible defaults target HELM Lite):
      HELM_GCS_BUCKET = crfm-helm-public
      HELM_GCS_PREFIX = lite/benchmark_output/runs
      HELM_GCS_SUITE  = v1.0.0
      HELM_GCS_LIMIT  = 200
    """
    import re
    bucket = os.environ.get("HELM_GCS_BUCKET", "crfm-helm-public")
    prefix = os.environ.get("HELM_GCS_PREFIX", "lite/benchmark_output/runs")
    suite  = os.environ.get("HELM_GCS_SUITE",  "v1.0.0")
    limit  = int(os.environ.get("HELM_GCS_LIMIT", "200"))

    # Debug holder
    if not hasattr(app.state, "helm_gcs_debug"):
        app.state.helm_gcs_debug = {}
    dbg = app.state.helm_gcs_debug
    dbg.clear()

    # 1) List objects
    list_url = f"https://storage.googleapis.com/storage/v1/b/{bucket}/o"
    params = {"prefix": f"{prefix}/{suite}/", "fields": "items(name),nextPageToken"}
    headers = {"User-Agent": "llm-router-ingestor/1.0"}

    def gcs_list_all(max_pages=20):
        out, token, pages = [], None, 0
        while pages < max_pages:
            p = dict(params)
            if token:
                p["pageToken"] = token
            r = requests.get(list_url, params=p, headers=headers, timeout=60)
            dbg.setdefault("list_calls", []).append({
                "status": r.status_code, "url": r.url, "first_120": r.text[:120]
            })
            if r.status_code != 200:
                break
            j = r.json()
            items = j.get("items") or []
            out.extend(items)
            token = j.get("nextPageToken")
            pages += 1
            if not token:
                break
        return out

    items = gcs_list_all(max_pages=20)
    if not items:
        dbg["note"] = "No items returned from GCS list"
        return []

    # 2) Group by run folder .../<SUITE>/<RUN_ID>/
    run_folders = {}
    for it in items:
        name = it.get("name", "")
        if not name.startswith(f"{prefix}/{suite}/"):
            continue
        parts = name.split("/")
        if len(parts) < 6:
            continue
        run_dir = "/".join(parts[:5]) + "/"  # up to RUN_ID/
        run_folders.setdefault(run_dir, []).append(name)

    dbg["runs_found"] = len(run_folders)

    rows = []
    processed = 0

    def dl_json(gcs_object_name: str):
        raw_url = f"https://storage.googleapis.com/{bucket}/{gcs_object_name}"
        r = requests.get(raw_url, headers=headers, timeout=60)
        dbg.setdefault("fetches", []).append({
            "status": r.status_code, "name": gcs_object_name, "first_120": r.text[:120]
        })
        if r.status_code != 200:
            return None
        try:
            return r.json()
        except Exception:
            return None

    for run_path, names in list(run_folders.items())[:limit]:
        run_spec = None
        stats = None
        for objname in names:
            if objname.endswith("/run_spec.json"):
                run_spec = dl_json(objname)
            elif objname.endswith("/stats.json"):
                stats = dl_json(objname)

        if not run_spec or not stats:
            continue

        # 3) Model id from run_spec
        model_id = None
        try:
            rs = run_spec.get("run_spec") or run_spec
            model_id = (rs.get("model")
                        or (rs.get("models") or [None])[0]
                        or (rs.get("adapter_spec") or {}).get("model"))
        except Exception:
            model_id = None

        # Fallback: derive from run_spec["name"] or run folder
        if not model_id:
            # Try name like "gsm:model=meta_llama-2-13b"
            name_field = (run_spec.get("name") or "")
            m = re.search(r"model=([^,\s]+)", name_field)
            if m:
                model_id = m.group(1)
        if not model_id:
            model_id = run_path.rstrip("/").split("/")[-1]

        if models and model_id not in models:
            processed += 1
            if processed >= limit:
                break
            continue

        # 4) Extract useful metrics from stats.json
        rows_added = 0

        def emit_metric(metric_name: str, val: float):
            nonlocal rows_added
            if val is None:
                return
            metric_name_l = (metric_name or "").strip().lower()
            if not metric_name_l:
                return
            task, diff = map_metric_to_task_diff(metric_name_l)
            item = {
                "model_id": model_id,
                "name": metric_name_l,
                "value": float(val),
                "unit": "score"
            }
            if task: item["task"] = task
            if diff: item["difficulty"] = diff
            rows.append(item)
            rows_added += 1

        # Optional scenario hint (unused but available)
        scenario_name = None
        try:
            scenario_name = (run_spec.get("name") or "").split(":")[0] or None
        except Exception:
            pass

        if isinstance(stats, list):
            for entry in stats:
                try:
                    nobj = entry.get("name") or {}
                    metric_name = (nobj.get("name") or "").strip().lower()
                    split = (nobj.get("split") or "").strip().lower()
                    if split and split != "test":
                        continue  # prefer test split
                    if not is_allowed_metric_name(metric_name):
                        continue  # drop housekeeping like num_references, prompt tokens, etc.

                    # choose a representative scalar (prefer mean)
                    val = None
                    for key in ("mean", "accuracy", "acc", "f1", "exact_match", "em", "score", "bleu", "sacrebleu",
                                "meteor", "chrf", "rouge1", "rouge2", "rougel", "auc", "roc_auc", "precision", "recall"):
                        v = entry.get(key)
                        if isinstance(v, (int, float)):
                            val = float(v); break
                    if val is None:
                        # fallback: derive from sum/count if present
                        s = entry.get("sum"); c = entry.get("count")
                        if isinstance(s, (int, float)) and isinstance(c, (int, float)) and c > 0:
                            val = float(s) / float(c)

                    emit_metric(metric_name, val)
                except Exception:
                    continue

        elif isinstance(stats, dict):
            def walk_metrics(obj, prefix=""):
                if isinstance(obj, dict):
                    for k, v in obj.items():
                        newp = f"{prefix}.{k}" if prefix else k
                        if isinstance(v, (int, float)):
                            yield (newp, float(v))
                        elif isinstance(v, dict):
                            yield from walk_metrics(v, newp)
                        elif isinstance(v, list):
                            for el in v:
                                if isinstance(el, dict):
                                    cand = (el.get("mean") or el.get("accuracy") or
                                            el.get("acc") or el.get("f1") or el.get("score"))
                                    if isinstance(cand, (int, float)):
                                        yield (newp, float(cand))
            for k, v in walk_metrics(stats):
                metric_name = k.rsplit(".", 1)[-1].lower()
                if any(s in metric_name for s in ["acc", "accuracy", "f1", "em", "exact_match", "score",
                                                  "auc", "mmlu", "gsm8k", "math", "bbh", "humaneval", "human_eval", "ifeval"]):
                    emit_metric(metric_name, v)

        processed += 1
        if processed >= limit:
            break

    dbg["processed_runs"] = processed
    dbg["emitted_rows"] = len(rows)
    return rows
@app.get("/healthz")
def healthz(): return {"ok": True}
@app.post("/refresh")
def refresh(req: RefreshRequest):
    sources = [s.strip().lower() for s in (req.sources or os.environ.get("SOURCES","open-llm-leaderboard,lmarena,artificialanalysis,helm,helm_gcs").split(",")) if s.strip()]
    models = req.models
    total = 0
    result_detail = {}

    for s in sources:
        rows = []
        src_debug = None
        try:
            if s == "open-llm-leaderboard":
                rows = ingest_open_llm_leaderboard(models)
                src_debug = getattr(app.state, "hf_debug", None)
            elif s == "lmarena":
                rows = ingest_lmarena(models)
                src_debug = getattr(app.state, "lmarena_debug", None)
            elif s == "helm":
                rows = ingest_helm(models)
            elif s == "helm_gcs":
                rows = ingest_helm_gcs(models)
                src_debug = getattr(app.state, "helm_gcs_debug", None)
            elif s == "artificialanalysis":
                rows = ingest_artificialanalysis_llms(models) or scrape_artificialanalysis_models_page(models)
        except Exception as e:
            result_detail[s] = {"error": str(e)}
            rows = []

        result_detail[s] = {
            "rows_found": len(rows),
            "sample": rows[:3] if rows else [],
            "fetch_debug": src_debug or {}
        }

        if rows:
            try:
                out = push(s, rows)
                result_detail[s]["pushed"] = out
                total += out.get("count", 0)
            except Exception as e:
                result_detail[s]["push_error"] = str(e)

    return {"ok": True, "ingested": total, "sources": sources, "detail": result_detail}
