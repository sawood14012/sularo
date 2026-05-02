package format

import (
	"encoding/json"
	"io"

	"github.com/sawood14012/sularo/internal/test"
)

type JSON struct{}

type jsonOutput struct {
	Total   int          `json:"total"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Skipped int          `json:"skipped"`
	Results []jsonResult `json:"results"`
}

type jsonResult struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Message    string  `json:"message,omitempty"`
	DurationMs float64 `json:"duration_ms"`
}

func (j JSON) Write(w io.Writer, results []test.Result) {
	out := jsonOutput{
		Total:   len(results),
		Results: make([]jsonResult, 0, len(results)),
	}

	for _, r := range results {
		switch r.Status {
		case test.StatusPass:
			out.Passed++
		case test.StatusFail:
			out.Failed++
		case test.StatusSkip:
			out.Skipped++
		}
		out.Results = append(out.Results, jsonResult{
			Name:       r.Name,
			Status:     string(r.Status),
			Message:    r.Message,
			DurationMs: float64(r.Duration.Milliseconds()),
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
