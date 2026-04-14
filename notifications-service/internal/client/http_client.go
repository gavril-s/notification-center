package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body any, headers map[string]string) ([]byte, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *HTTPClient) doRequestJSON(ctx context.Context, method, path string, body any, result any, headers map[string]string) error {
	respBody, err := c.doRequest(ctx, method, path, body, headers)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

// Recipients service client methods

type ResolveRecipientRequest struct {
	ContactID *string `json:"contact_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

type ResolveRecipientResponse struct {
	ContactID string `json:"contact_id"`
	UserID    string `json:"user_id"`
	Valid     bool   `json:"valid"`
}

func (c *HTTPClient) ResolveRecipient(ctx context.Context, req ResolveRecipientRequest, traceID string) (*ResolveRecipientResponse, error) {
	var resp ResolveRecipientResponse
	err := c.doRequestJSON(ctx, "POST", "/internal/recipients/resolve", req, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type GetUserContactsResponse struct {
	Contacts []Contact `json:"contacts"`
}

type Contact struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

func (c *HTTPClient) GetUserContacts(ctx context.Context, userID, traceID string) (*GetUserContactsResponse, error) {
	var resp GetUserContactsResponse
	err := c.doRequestJSON(ctx, "GET", fmt.Sprintf("/internal/recipients/users/%s/contacts", userID), nil, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Sources service client methods

type Template struct {
	ID        string   `json:"id"`
	Channel   string   `json:"channel"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	Variables []string `json:"variables"`
}

func (c *HTTPClient) GetTemplate(ctx context.Context, templateID, traceID string) (*Template, error) {
	var resp Template
	err := c.doRequestJSON(ctx, "GET", fmt.Sprintf("/internal/sources/templates/%s", templateID), nil, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type Campaign struct {
	ID          string  `json:"id"`
	SenderID    string  `json:"sender_id"`
	TemplateID  string  `json:"template_id"`
	GroupID     string  `json:"group_id"`
	ScheduledAt *string `json:"scheduled_at"`
	Status      string  `json:"status"`
}

func (c *HTTPClient) GetCampaign(ctx context.Context, campaignID, traceID string) (*Campaign, error) {
	var resp Campaign
	err := c.doRequestJSON(ctx, "GET", fmt.Sprintf("/internal/sources/campaigns/%s", campaignID), nil, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type Sender struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Credentials string `json:"credentials"`
}

func (c *HTTPClient) ResolveCredential(ctx context.Context, integrationKey, traceID string) (*Sender, error) {
	var resp Sender
	err := c.doRequestJSON(ctx, "POST", "/internal/sources/senders/resolve-credential", map[string]string{"integration_key": integrationKey}, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) CheckOperatorPermission(ctx context.Context, senderID, userID, traceID string) (bool, error) {
	err := c.doRequestJSON(ctx, "GET", fmt.Sprintf("/internal/sources/senders/%s/operators/%s", senderID, userID), nil, nil, map[string]string{"X-Trace-ID": traceID})
	return err == nil, nil
}

type GroupMember struct {
	UserID    string `json:"user_id"`
	ContactID string `json:"contact_id"`
}

type GetGroupMembersResponse struct {
	Members []GroupMember `json:"members"`
}

func (c *HTTPClient) GetGroupMembers(ctx context.Context, groupID, traceID string) ([]GroupMember, error) {
	var resp GetGroupMembersResponse
	err := c.doRequestJSON(ctx, "GET", fmt.Sprintf("/internal/sources/groups/%s/members", groupID), nil, &resp, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return resp.Members, nil
}
