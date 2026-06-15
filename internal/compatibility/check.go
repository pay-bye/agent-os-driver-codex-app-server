package compatibility

import (
	"context"
	"strconv"
	"strings"
)

type InvocationMetadataReader interface {
	Metadata(context.Context) (Metadata, error)
}

type AppMetadataReader interface {
	Metadata(context.Context) (AppMetadata, error)
}

type Verifier struct {
	Requirements Requirements
	Invocation   InvocationMetadataReader
	App          AppMetadataReader
}

func (v Verifier) Verify(ctx context.Context) (Result, error) {
	if err := v.Requirements.Validate(); err != nil {
		return failure(err)
	}
	app, err := v.App.Metadata(ctx)
	if err != nil {
		return failure(err)
	}
	if err := v.Requirements.CheckApp(app); err != nil {
		return failure(err)
	}
	metadata, err := v.Invocation.Metadata(ctx)
	if err != nil {
		return failure(err)
	}
	if err := v.Requirements.CheckMetadata(metadata); err != nil {
		return failure(err)
	}
	return Result{
		Requirements:       v.Requirements,
		InvocationMetadata: metadata,
		AppServerDigest:    app.SchemaDigest,
		DiagnosticCode:     Code(nil),
	}, nil
}

func failure(err error) (Result, error) {
	return Result{DiagnosticCode: Code(err)}, err
}

func versionBounds(values []string) (int, int) {
	minimum, maximum := 0, 0
	for index, value := range values {
		number, err := versionNumber(value)
		if err != nil {
			continue
		}
		if index == 0 || number < minimum {
			minimum = number
		}
		if index == 0 || number > maximum {
			maximum = number
		}
	}
	return minimum, maximum
}

func versionNumber(value string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(value, "v"))
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsAll(values []string, wanted []string) bool {
	for _, value := range wanted {
		if !contains(values, value) {
			return false
		}
	}
	return true
}
