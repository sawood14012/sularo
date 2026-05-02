package test

import "time"

type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

type Result struct {
	Name     string
	Status   Status
	Message  string // diff or error detail on failure
	Duration time.Duration
}
