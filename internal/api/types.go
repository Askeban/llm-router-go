package api

type RecommendRequest struct {
	Prompt  string         `json:"prompt"`
	Context *Context       `json:"context,omitempty"`
	Catalog []string       `json:"catalog,omitempty"`
	Limits  *Limits        `json:"limits,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type Context struct {
	Language       string            `json:"language,omitempty"`
	FilePath       string            `json:"file_path,omitempty"`
	SelectionBytes int               `json:"selection_bytes,omitempty"`
	Repo           *RepoInfo         `json:"repo,omitempty"`
	Signals        map[string]string `json:"signals,omitempty"`
	Snippets       []ContextSnippet  `json:"snippets,omitempty"`
}

type ContextSnippet struct {
	Source    string `json:"source"`
	Path      string `json:"path,omitempty"`
	Text      string `json:"text,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
}

type RepoInfo struct {
	Name        string `json:"name,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	Monorepo    bool   `json:"monorepo,omitempty"`
	LocEstimate int    `json:"loc_estimate,omitempty"`
}

type Limits struct {
	MaxPromptKB int `json:"max_prompt_kb,omitempty"`
}

type RecommendResponse struct {
	RecommendedModel  string   `json:"recommended_model"`
	Rationale         string   `json:"rationale,omitempty"`
	Confidence        float32  `json:"confidence,omitempty"`
	Alternatives      []string `json:"alternatives,omitempty"`
	CostMsEstimate    int      `json:"cost_ms_estimate,omitempty"`
	TokensInEstimate  int      `json:"tokens_in_estimate,omitempty"`
	TokensOutEstimate int      `json:"tokens_out_estimate,omitempty"`
	Flags             []string `json:"flags,omitempty"`
	TraceID           string   `json:"trace_id,omitempty"`
	Version           string   `json:"version"`
}
