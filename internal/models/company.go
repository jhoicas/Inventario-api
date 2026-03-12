package models

import "time"

type Resolution struct {
	ID               string
	CompanyID        string
	Prefix           string
	ResolutionNumber string
	FromNumber       int64
	ToNumber         int64
	ValidFrom        time.Time
	ValidUntil       time.Time
	Environment      string
	UsedNumbers      int64
}

func (r Resolution) AlertThreshold() bool {
	total := r.ToNumber - r.FromNumber + 1
	if total <= 0 {
		return false
	}

	used := r.UsedNumbers
	if used < 0 {
		used = 0
	}
	if used > total {
		used = total
	}

	available := total - used
	return float64(available) < float64(total)*0.10
}
