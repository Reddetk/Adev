package domain

import "time"

// Source is how the request reached adev.
type Source string

const (
	SourceOperator Source = "operator"
	SourceAPI      Source = "api"
	SourceIssue    Source = "issue"
)

// Locale is the request language.
type Locale string

const (
	LocaleEN Locale = "en"
	LocaleRU Locale = "ru"
)

// UserInput is the raw operator request. It is stored so that later inputs are complete.
type UserInput struct {
	RawRequest  Text   `json:"rawRequest"`
	Attachments []Ref  `json:"attachments,omitempty"`
	Source      Source `json:"source"`
	ReceivedAt  string `json:"receivedAt"`
	Locale      Locale `json:"locale,omitempty"`
}

// ParseReceivedAt returns the receive time, or the zero time when the value is absent.
func (u UserInput) ParseReceivedAt() (time.Time, error) {
	if u.ReceivedAt == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, u.ReceivedAt)
}
