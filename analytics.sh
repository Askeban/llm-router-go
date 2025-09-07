export PARALLEL_API_KEY="MZj70pAIpUcy7oovQJyPFQtVDHElXxrDgMA-z143"

# Create a task run
RUN_JSON=$(curl -s 'https://api.parallel.ai/v1/tasks/runs' \
  -H "x-api-key: ${PARALLEL_API_KEY}" \
  -H 'Content-Type: application/json' \
  -d '{
    "processor": "base",
    "input": {
      "source_name": "huggingface_open_llm_leaderboard",
      "url": "https://huggingface.co/spaces/open-llm-leaderboard/open_llm_leaderboard",
      "limit": 100,
      "notes": "Extract top models and common benchmarks (MMLU/MMLU-Pro, GSM8K, HellaSwag, TruthfulQA, BBH, GPQA). Include license, open_source, context window, eval date if present."
    },
    "task_spec": {
      "output_schema": {
        "type": "json",
        "json_schema": {
          "type": "object",
          "properties": {
            "source":       { "type": "string" },
            "source_url":   { "type": "string" },
            "scraped_at":   { "type": "string" },
            "models": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "model_display_name":       { "type": "string" },
                  "model_org":                { "type": "string" },
                  "model_id":                 { "type": "string" },
                  "license":                  { "type": "string" },
                  "open_source":              { "type": "boolean" },
                  "context_window_tokens":    { "type": "integer" },
                  "benchmarks": {
                    "type": "object",
                    "properties": {
                      "MMLU":        { "type": "number" },
                      "MMLU_Pro":    { "type": "number" },
                      "GSM8K":       { "type": "number" },
                      "HellaSwag":   { "type": "number" },
                      "TruthfulQA":  { "type": "number" },
                      "BBH":         { "type": "number" },
                      "GPQA":        { "type": "number" }
                    },
                    "additionalProperties": true
                  },
                  "eval_date": { "type": "string" },
                  "model_page":{ "type": "string" }
                },
                "required": ["model_display_name"]
              }
            }
          },
          "required": ["source","source_url","scraped_at","models"],
          "additionalProperties": false
        }
      }
    }
  }')

# Capture run id
RUN_ID=$(echo "$RUN_JSON" | jq -r ".run_id")
echo "RUN_ID=$RUN_ID"

# Fetch the structured result (poll until ready if needed)
curl -s "https://api.parallel.ai/v1/tasks/runs/${RUN_ID}/result" \
  -H "x-api-key: ${PARALLEL_API_KEY}" | jq .

