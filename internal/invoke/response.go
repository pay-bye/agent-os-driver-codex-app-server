package invoke

import (
	"fmt"
	"strings"
	"time"
)

type Claim struct {
	Empty      bool
	LeaseID    string
	Token      string
	WorkItemID string
	Payload    Payload
	ExpiresAt  time.Time
}

type claimResponse struct {
	Empty      bool    `json:"empty"`
	LeaseID    string  `json:"lease_id"`
	Token      string  `json:"lease_token"`
	WorkItemID string  `json:"work_item_id"`
	Payload    Payload `json:"payload"`
	ExpiresAt  string  `json:"expires_at"`
}

func claimFromResponse(response claimResponse) (Claim, error) {
	if response.Empty {
		return Claim{Empty: true}, nil
	}
	if strings.TrimSpace(response.Token) == "" {
		return Claim{}, fmt.Errorf("invalid_claim_token")
	}
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return Claim{}, fmt.Errorf("invalid_claim_expires_at: %w", err)
	}
	return Claim{
		LeaseID:    response.LeaseID,
		Token:      response.Token,
		WorkItemID: response.WorkItemID,
		Payload:    response.Payload,
		ExpiresAt:  expiresAt,
	}, nil
}
