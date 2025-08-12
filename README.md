# llm-router-go

This service recommends suitable language models for a given prompt and context.

## Copilot Extension Integration

- Endpoint: `POST /v1/recommend`
- OpenAPI: `api/openapi.yaml`
- Auth: header `X-API-Key` (set `ROUTER_API_KEY`)
- **Note:** Calls made from Copilot Chat consume a Copilot request (billing by GitHub).
- No public API to auto-switch Copilot’s model; we return the recommendation for the client UI to guide the user.

