package invoke

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEndpointUnavailable = errors.New("endpoint_unavailable")
	ErrMetadataMalformed   = errors.New("metadata_malformed")
)

type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type Metadata struct {
	ContractVersion string   `json:"contract_version"`
	SchemaSetDigest string   `json:"schema_set_digest"`
	Features        []string `json:"features"`
	Routes          []Route  `json:"routes"`
}

func (m Metadata) Validate() error {
	if err := ValidateRoutes(m.Routes); err != nil {
		return err
	}
	routes := PublishedRoutes()
	if len(m.Routes) != len(routes) {
		return malformed(fmt.Errorf("route count=%d", len(m.Routes)))
	}
	for _, route := range routes {
		if !containsRoute(m.Routes, route) {
			return malformed(fmt.Errorf("route missing=%s %s", route.Method, route.Path))
		}
	}
	return nil
}

func ValidateRoutes(values []Route) error {
	if len(values) == 0 {
		return malformed(nil)
	}
	seen := make(map[Route]bool, len(values))
	for _, value := range values {
		if value.Method != strings.ToUpper(value.Method) || !strings.HasPrefix(value.Path, "/") || !isPublishedRoute(value) {
			return malformed(fmt.Errorf("route=%s %s", value.Method, value.Path))
		}
		if seen[value] {
			return malformed(fmt.Errorf("duplicate route=%s %s", value.Method, value.Path))
		}
		seen[value] = true
	}
	return nil
}

func PublishedRoutes() []Route {
	return []Route{
		{Method: "POST", Path: "/claim"},
		{Method: "POST", Path: "/ack"},
		{Method: "POST", Path: "/nack"},
		{Method: "POST", Path: "/extend"},
		{Method: "GET", Path: "/compatibility"},
	}
}

func ContainsRoutes(values []Route, wanted []Route) bool {
	for _, value := range wanted {
		if !containsRoute(values, value) {
			return false
		}
	}
	return true
}

func containsRoute(values []Route, wanted Route) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func isPublishedRoute(value Route) bool {
	return containsRoute(PublishedRoutes(), value)
}

func malformed(err error) error {
	if err == nil {
		return ErrMetadataMalformed
	}
	return fmt.Errorf("%w: %v", ErrMetadataMalformed, err)
}
