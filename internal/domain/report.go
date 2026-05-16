package domain

import (
	"errors"
	"time"
)

var (
	ErrReportNotFound = errors.New("report not found")
	ErrInvalidID      = errors.New("invalid report id")
)

type Analysis struct {
	Components      []string `json:"components"      bson:"components"`
	Risks           []string `json:"risks"           bson:"risks"`
	Recommendations []string `json:"recommendations" bson:"recommendations"`
}

type Report struct {
	ID          string
	ProcessID   string
	Analysis    Analysis
	RawResponse string
	CreatedAt   time.Time
}
