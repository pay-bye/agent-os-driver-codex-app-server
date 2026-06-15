package invoke

import (
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
)

type claimRequest struct {
	Channel string `json:"channel_key"`
	Lease   string `json:"lease_id"`
	Seconds int    `json:"lease_seconds"`
}

type extendRequest struct {
	Lease     string `json:"lease_id"`
	Token     string `json:"lease_token"`
	ExpiresAt string `json:"requested_expires_at"`
}

type completionRequest struct {
	Lease string        `json:"lease_id"`
	Token string        `json:"lease_token"`
	Needs []config.Need `json:"declared_needs"`
}

type failureRequest struct {
	Lease   string        `json:"lease_id"`
	Token   string        `json:"lease_token"`
	Failure Payload       `json:"failure_payload"`
	Needs   []config.Need `json:"declared_needs"`
}
