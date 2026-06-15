package status

func FailurePayload(code string, _ ...map[string]any) map[string]any {
	return map[string]any{"error_code": code}
}
