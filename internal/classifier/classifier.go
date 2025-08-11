package classifier

import (
    "regexp"
    "strings"

    "llm-router-go/internal/types"
)

// commonCodeKeywords is a list of substrings that frequently appear in code.
var commonCodeKeywords = []string{
    "def", "class", "import", "#include", "public", "private", "function",
}

// Classify returns "code" if the prompt or its context resembles source code, or
// "text" otherwise.  It uses heuristics based on keywords, punctuation and file
// extensions.  A nil context is treated as empty.
func Classify(prompt string, ctx *types.Context) string {
    // Search for keywords in prompt
    lower := strings.ToLower(prompt)
    for _, kw := range commonCodeKeywords {
        if strings.Contains(lower, kw) {
            return "code"
        }
    }
    // Punctuation patterns typical of code: braces, semicolons
    if regexp.MustCompile(`[{};]`).MatchString(prompt) {
        return "code"
    }
    // Inspect context files
    if ctx != nil {
        for _, f := range ctx.Files {
            // Extension check
            for _, ext := range []string{".py", ".js", ".ts", ".java", ".cpp", ".c", ".cs", ".go", ".rb"} {
                if strings.HasSuffix(strings.ToLower(f.Path), ext) {
                    return "code"
                }
            }
            contentLower := strings.ToLower(f.Content)
            for _, kw := range commonCodeKeywords {
                if strings.Contains(contentLower, kw) {
                    return "code"
                }
            }
            if regexp.MustCompile(`[{};]`).MatchString(f.Content) {
                return "code"
            }
        }
    }
    return "text"
}