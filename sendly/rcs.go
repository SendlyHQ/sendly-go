package sendly

import "context"

// RCSService is the RCS channel: agent discovery and recipient capability
// pre-flight. Send RCS messages with MessagesService.SendRcs.
//
// RCS messages come from a verified, branded agent rather than a bare
// phone number. Agents are registered per brand by Sendly support —
// contact support to set one up. RCS availability is rolling out
// gradually; while it's off for an account, these endpoints return 404.
type RCSService struct {
	client *Client
	// Agents lists the workspace's RCS agents.
	Agents *RCSAgentsService
}

// RCSAgentsService lists the workspace's RCS agents.
type RCSAgentsService struct {
	client *Client
}

// RcsAgent is an RCS agent registered for the workspace's brand.
type RcsAgent struct {
	// ID is the unique agent identifier — pass it as AgentID when sending.
	ID string `json:"id"`
	// Name is the agent name recipients see.
	Name string `json:"name"`
	// Status is the agent's lifecycle status; "testing" and "approved"
	// agents can send ("testing" reaches invited test numbers only).
	Status string `json:"status"`
	// UseCase is the agent's declared use case, nil when not set.
	UseCase *string `json:"useCase,omitempty"`
	// Sendable is true when the agent is fully provisioned and its status
	// allows sending.
	Sendable bool `json:"sendable"`
	// CreatedAt is the ISO 8601 timestamp when the agent was registered.
	CreatedAt string `json:"createdAt"`
}

// RcsAgentListResponse wraps the workspace's RCS agents.
type RcsAgentListResponse struct {
	Agents []RcsAgent `json:"agents"`
}

// RcsCapability reports whether a recipient can receive RCS.
type RcsCapability struct {
	// To is the checked number, in E.164 format.
	To string `json:"to"`
	// AgentID is the agent the check ran against.
	AgentID string `json:"agentId"`
	// Capable is true when the recipient's device and network support RCS.
	Capable bool `json:"capable"`
	// Features are the RCS features the recipient supports.
	Features []string `json:"features"`
}

// List returns the workspace's RCS agents, newest first, with lifecycle
// status and sendability. An empty list means no agent is registered yet —
// contact support to set one up for your brand.
func (s *RCSAgentsService) List(ctx context.Context) (*RcsAgentListResponse, error) {
	var resp RcsAgentListResponse
	if err := s.client.request(ctx, "GET", "/rcs/agents", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Capability checks whether a recipient (E.164) can receive RCS before
// sending. agentID picks the agent to check against; pass "" when the
// workspace has exactly one agent. Requires a live API key.
//
// This pre-flight is optional — text sends fall back to SMS on their own
// when the recipient isn't RCS-capable.
func (s *RCSService) Capability(ctx context.Context, to, agentID string) (*RcsCapability, error) {
	if to == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}

	path := "/rcs/capability" + buildQueryString(map[string]string{"to": to, "agentId": agentID})
	var resp RcsCapability
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
