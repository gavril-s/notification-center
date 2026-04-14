package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClient_GetUserContacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/recipients/users/user-1/contacts" {
			t.Errorf("expected path /internal/recipients/users/user-1/contacts, got %s", r.URL.Path)
		}
		traceID := r.Header.Get("X-Trace-ID")
		if traceID != "test-trace" {
			t.Errorf("expected X-Trace-ID test-trace, got %s", traceID)
		}

		resp := GetUserContactsResponse{
			Contacts: []Contact{
				{ID: "contact-1", Type: "email", Value: "test@example.com", Primary: true},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	resp, err := client.GetUserContacts(context.Background(), "user-1", "test-trace")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(resp.Contacts))
	}
	if resp.Contacts[0].ID != "contact-1" {
		t.Errorf("expected contact ID contact-1, got %s", resp.Contacts[0].ID)
	}
}

func TestHTTPClient_GetTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/sources/templates/template-1" {
			t.Errorf("expected path /internal/sources/templates/template-1, got %s", r.URL.Path)
		}

		resp := Template{
			ID:      "template-1",
			Channel: "email",
			Subject: "Test Subject",
			Body:    "Test Body",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	resp, err := client.GetTemplate(context.Background(), "template-1", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != "template-1" {
		t.Errorf("expected template ID template-1, got %s", resp.ID)
	}
	if resp.Channel != "email" {
		t.Errorf("expected channel email, got %s", resp.Channel)
	}
}

func TestHTTPClient_GetCampaign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/sources/campaigns/campaign-1" {
			t.Errorf("expected path /internal/sources/campaigns/campaign-1, got %s", r.URL.Path)
		}

		resp := Campaign{
			ID:       "campaign-1",
			SenderID: "sender-1",
			GroupID:  "group-1",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	resp, err := client.GetCampaign(context.Background(), "campaign-1", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != "campaign-1" {
		t.Errorf("expected campaign ID campaign-1, got %s", resp.ID)
	}
}

func TestHTTPClient_ResolveCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/sources/senders/resolve-credential" {
			t.Errorf("expected path /internal/sources/senders/resolve-credential, got %s", r.URL.Path)
		}

		resp := Sender{
			ID:   "sender-1",
			Name: "Test Sender",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	resp, err := client.ResolveCredential(context.Background(), "integration-key", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != "sender-1" {
		t.Errorf("expected sender ID sender-1, got %s", resp.ID)
	}
}

func TestHTTPClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	_, err := client.GetTemplate(context.Background(), "template-1", "trace-1")

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestHTTPClient_GetGroupMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/sources/groups/group-1/members" {
			t.Errorf("expected path /internal/sources/groups/group-1/members, got %s", r.URL.Path)
		}

		resp := GetGroupMembersResponse{
			Members: []GroupMember{
				{UserID: "user-1", ContactID: "contact-1"},
				{UserID: "user-2", ContactID: "contact-2"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	resp, err := client.GetGroupMembers(context.Background(), "group-1", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 members, got %d", len(resp))
	}
}

func TestHTTPClient_CheckOperatorPermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/sources/senders/sender-1/operators/user-1" {
			t.Errorf("expected path /internal/sources/senders/sender-1/operators/user-1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	hasPermission, err := client.CheckOperatorPermission(context.Background(), "sender-1", "user-1", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasPermission {
		t.Error("expected true, got false")
	}
}

func TestHTTPClient_CheckOperatorPermission_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	hasPermission, _ := client.CheckOperatorPermission(context.Background(), "sender-1", "user-1", "trace-1")

	if hasPermission {
		t.Error("expected false, got true")
	}
}

func TestHTTPClient_ResolveRecipient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/recipients/resolve" {
			t.Errorf("expected path /internal/recipients/resolve, got %s", r.URL.Path)
		}

		resp := ResolveRecipientResponse{
			ContactID: "contact-1",
			UserID:    "user-1",
			Valid:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	contactID := "contact-1"
	resp, err := client.ResolveRecipient(context.Background(), ResolveRecipientRequest{ContactID: &contactID}, "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ContactID != "contact-1" {
		t.Errorf("expected contact ID contact-1, got %s", resp.ContactID)
	}
	if !resp.Valid {
		t.Error("expected valid to be true")
	}
}
