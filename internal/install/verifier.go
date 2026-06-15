package install

import (
	"context"
	_ "embed"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/compatibility"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/invoke"
)

type liveVerifier struct {
	Env []string
}

func (v liveVerifier) Verify(ctx context.Context, item config.Config) (compatibility.Result, error) {
	requirements, err := DefaultRequirements()
	if err != nil {
		return compatibility.Result{DiagnosticCode: compatibility.Code(err)}, err
	}
	return compatibility.Verifier{
		Requirements: requirements,
		Invocation:   invoke.New(item.InvocationBaseURL, nil),
		App: compatibility.CommandAppMetadata{
			CodexBin: item.CodexBin,
			Env:      v.Env,
		},
	}.Verify(ctx)
}
