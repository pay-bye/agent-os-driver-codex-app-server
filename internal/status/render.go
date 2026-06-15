package status

import (
	"fmt"
	"strings"
)

func Render(counts Counts) string {
	var builder strings.Builder
	writeField(&builder, "source", counts.Source)
	writeField(&builder, "install_id", counts.InstallID)
	writeField(&builder, "driver_version", counts.DriverVersion)
	writeField(&builder, "app_version", counts.AppVersion)
	writeField(&builder, "config_digest", counts.ConfigDigest)
	writeNumber(&builder, "claim_attempts", counts.ClaimAttempts)
	writeNumber(&builder, "empty_claims", counts.EmptyClaims)
	writeField(&builder, "active_lease_id", counts.ActiveLeaseID)
	writeField(&builder, "work_item_id", counts.WorkItemID)
	writeField(&builder, "thread_id", counts.ThreadID)
	writeField(&builder, "turn_id", counts.TurnID)
	writeNumber(&builder, "ack_count", counts.Acks)
	writeNumber(&builder, "nack_count", counts.Nacks)
	writeNumber(&builder, "extension_count", counts.Extensions)
	writeField(&builder, "error_code", counts.LastErrorCode)
	return strings.TrimRight(builder.String(), "\n")
}

func RenderDiagnostic(source string, code string) string {
	return Render(Counts{Source: source, LastErrorCode: code})
}

func writeField(builder *strings.Builder, name string, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(builder, "%s=%s\n", name, value)
}

func writeNumber(builder *strings.Builder, name string, value int) {
	fmt.Fprintf(builder, "%s=%d\n", name, value)
}
