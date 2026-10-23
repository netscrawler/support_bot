package models

import (
	"fmt"
	"time"
)

type PeriodName string

const (
	PeriodToday     PeriodName = "today"
	PeriodYesterday PeriodName = "yesterday"
	PeriodWeek      PeriodName = "week"
	PeriodMonth     PeriodName = "month"
)

var periodCodes = map[PeriodName]string{
	PeriodToday:     "t",
	PeriodYesterday: "y",
	PeriodWeek:      "w",
	PeriodMonth:     "m",
}

// ParsePeriod: пустая строка означает today.
func ParsePeriod(s string) (PeriodName, error) {
	if s == "" {
		return PeriodToday, nil
	}

	p := PeriodName(s)
	if _, ok := periodCodes[p]; !ok {
		return "", fmt.Errorf("unknown period %q", s)
	}

	return p, nil
}

// Code возвращает короткий код для callback data.
func (p PeriodName) Code() string { return periodCodes[p] }

func PeriodByCode(code string) (PeriodName, bool) {
	for p, c := range periodCodes {
		if c == code {
			return p, true
		}
	}

	return "", false
}

// Range возвращает полуинтервал [from, to) в часовом поясе now.
func (p PeriodName) Range(now time.Time) (from, to time.Time, err error) {
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch p {
	case PeriodToday:
		return d, d.AddDate(0, 0, 1), nil
	case PeriodYesterday:
		return d.AddDate(0, 0, -1), d, nil
	case PeriodWeek:
		return d.AddDate(0, 0, -7), d, nil
	case PeriodMonth:
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		return first.AddDate(0, -1, 0), first, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unknown period %q", string(p))
	}
}
