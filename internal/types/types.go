package types

// File represents a single file in the prompt context.  It mirrors the
// structure expected in the API request payload.
type File struct {
    Path    string `json:"path"`
    Content string `json:"content"`
}

// Context holds an optional list of files.  Additional fields could be
// introduced in the future (e.g., repository metadata) without changing
// existing callers.
type Context struct {
    Files []File `json:"files"`
}