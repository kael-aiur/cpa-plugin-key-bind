package plugin

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestManagementRegistrationKeepsOnlyWebResource(t *testing.T) {
	registration := NewApp().managementRegistration()
	if len(registration.Routes) != 0 {
		t.Fatalf("expected no management API routes, got %#v", registration.Routes)
	}
	if len(registration.Resources) != 1 {
		t.Fatalf("expected one web resource, got %#v", registration.Resources)
	}
	resource := registration.Resources[0]
	if resource.Path == "" || resource.Menu == "" {
		t.Fatalf("unexpected web resource: %#v", resource)
	}
}

func TestManagementBindsGETReturnsNotFound(t *testing.T) {
	raw, err := json.Marshal(ManagementRequest{
		Method: http.MethodGet,
		Path:   mgmtBase + "/binds",
	})
	if err != nil {
		t.Fatalf("marshal management request: %v", err)
	}

	response, err := NewApp().handleManagement(raw)
	if err != nil {
		t.Fatalf("handleManagement: %v", err)
	}
	var envelope Envelope
	if err := json.Unmarshal(response, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected transport envelope success, got error: %+v", envelope.Error)
	}
	var management ManagementResponse
	if err := json.Unmarshal(envelope.Result, &management); err != nil {
		t.Fatalf("unmarshal management response: %v", err)
	}
	if management.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", management.StatusCode)
	}
}
