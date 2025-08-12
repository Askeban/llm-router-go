package api

import "errors"

func (r *RecommendRequest) Validate() error {
	if r.Prompt == "" {
		return errors.New("prompt is required")
	}
	if r.Limits != nil && r.Limits.MaxPromptKB < 0 {
		return errors.New("limits.max_prompt_kb must be >= 0")
	}
	return nil
}
