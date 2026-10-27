package domain

type ModelCall struct {
	ID        uint    `json:"id"`
	RunID     int     `json:"runid" validate:"required,gte=0"`
	Profile   string  `json:"profile" validate:"required"`
	Model     string  `json:"model" validate:"required"`
	TokensIn  int     `json:"tokensin" validate:"gte=0"`
	TokensOut int     `json:"tokensout" validate:"gte=0"`
	Cost      float64 `json:"cost" validate:"gte=0"`
	LatencyMs int     `json:"latencyms" validate:"gte=0"`
	// The dialogue of the attempt: which stage it served, the full prompt the
	// agent sent, the raw answer, and the gate verdicts that rejected it.
	Stage    string `json:"stage"`
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
	Errors   string `json:"errors,omitempty"`
}

func (m *ModelCall) Validate() error {
	if m.RunID < 0 {
		return ErrInvalidModelCallRunID
	}
	if m.Profile == "" {
		return ErrInvalidModelCallProfile
	}
	if m.Model == "" {
		return ErrInvalidModelCallModel
	}
	if m.TokensIn < 0 {
		return ErrInvalidModelCallTokensIn
	}
	if m.TokensOut < 0 {
		return ErrInvalidModelCallTokensOut
	}
	if m.Cost < 0 {
		return ErrInvalidModelCallCost
	}
	if m.LatencyMs < 0 {
		return ErrInvalidModelCallLatencyMs
	}
	return nil
}
