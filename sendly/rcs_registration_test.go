package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const rcsBrandJSON = `{
	"id": "rcsb_1",
	"reviewStatus": "draft",
	"customerStage": "draft",
	"displayName": "Acme",
	"legalName": "Acme Holdings LLC",
	"legalEntityType": "LIMITED_LIABILITY_COMPANY",
	"organizationType": "PRIVATE_PROFIT",
	"stockSymbol": null,
	"websiteUrl": "https://example.com",
	"ein": "12-3456789",
	"address": {"line1": "1 Main St", "line2": null, "city": "Austin", "state": "TX", "postalCode": "78701", "countryCode": "US"},
	"contact": {"firstName": "Ada", "lastName": "Lovelace", "title": null, "email": "ada@example.com", "phoneNumber": "+15551234567"},
	"reviewNote": null,
	"rejectionReason": null,
	"submittedForReviewAt": null,
	"sentToCarrierAt": null,
	"verifiedAt": null,
	"createdAt": "2026-07-01T00:00:00Z",
	"updatedAt": "2026-07-01T00:00:00Z"
}`

const rcsAgentJSON = `{
	"id": "rcsa_1",
	"brandId": "rcsb_1",
	"status": "draft",
	"reviewStatus": "draft",
	"customerStage": "draft",
	"displayName": "Acme Support",
	"useCase": "MULTI_USE",
	"hostingRegion": "NORTH_AMERICA",
	"basics": {
		"displayName": "Acme Support",
		"useCase": "MULTI_USE",
		"hostingRegion": "NORTH_AMERICA",
		"description": "Order updates and support",
		"logoUrl": "https://example.com/logo.png",
		"brandColor": "#FF5500",
		"phoneNumber": {"number": "+15551234567", "label": "Support"},
		"website": {"url": "https://example.com", "label": "Site"},
		"email": null
	},
	"campaign": {
		"agentOverview": "Sends order updates",
		"interactions": [{"interactionType": "TRANSACTIONAL_UPDATES", "description": "Shipping"}],
		"messageExamples": ["Your order shipped", "Your order arrived", "Reply STOP to opt out"],
		"consentSettings": {"optInMethods": [{"methodType": "WEBSITE", "description": "Checkout"}], "doubleOptIn": false}
	},
	"testing": {"testUrl": "https://example.com/test", "messageId": null, "additionalInformation": null},
	"reviewNote": null,
	"rejectionReason": null,
	"testDevices": [{"id": "rcsd_1", "phoneNumber": "+15557654321", "label": "Ada", "inviteStatus": "PENDING", "createdAt": "2026-07-02T00:00:00Z"}],
	"submittedForReviewAt": null,
	"basicsSubmittedAt": null,
	"launchSubmittedAt": null,
	"liveAt": null,
	"createdAt": "2026-07-01T00:00:00Z",
	"updatedAt": "2026-07-02T00:00:00Z"
}`

func decodeRequestBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode request: %v", err)
	}
	return body
}

func TestRCSRegistrationGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/registration" {
			t.Errorf("expected path '/rcs/registration', got '%s'", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be 'Bearer test-api-key', got '%s'", auth)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"brand": ` + rcsBrandJSON + `,
			"agent": ` + rcsAgentJSON + `,
			"devices": [{"id": "rcsd_1", "phoneNumber": "+15557654321", "label": "Ada", "inviteStatus": "PENDING", "createdAt": "2026-07-02T00:00:00Z"}],
			"stage": "testing",
			"usEligible": true
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.RCS.Registration.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Stage != RcsCustomerStageTesting {
		t.Errorf("expected Stage to be 'testing', got '%s'", resp.Stage)
	}
	if !resp.USEligible {
		t.Error("expected USEligible to be true")
	}
	if resp.Brand == nil || resp.Brand.ID != "rcsb_1" {
		t.Fatalf("expected Brand rcsb_1, got %v", resp.Brand)
	}
	if resp.Brand.Address.CountryCode != "US" || resp.Brand.Address.Line2 != "" {
		t.Errorf("unexpected Address: %+v", resp.Brand.Address)
	}
	if resp.Brand.Contact.Email != "ada@example.com" {
		t.Errorf("expected Contact.Email to be 'ada@example.com', got '%s'", resp.Brand.Contact.Email)
	}
	if resp.Brand.StockSymbol != nil || resp.Brand.VerifiedAt != nil {
		t.Errorf("expected nil StockSymbol and VerifiedAt, got %v %v", resp.Brand.StockSymbol, resp.Brand.VerifiedAt)
	}
	if resp.Agent == nil || resp.Agent.ID != "rcsa_1" {
		t.Fatalf("expected Agent rcsa_1, got %v", resp.Agent)
	}
	if resp.Agent.BrandID == nil || *resp.Agent.BrandID != "rcsb_1" {
		t.Errorf("expected BrandID to be 'rcsb_1', got %v", resp.Agent.BrandID)
	}
	if resp.Agent.Basics.PhoneNumber == nil || resp.Agent.Basics.PhoneNumber.Number != "+15551234567" {
		t.Errorf("unexpected Basics.PhoneNumber: %v", resp.Agent.Basics.PhoneNumber)
	}
	if resp.Agent.Basics.Email != nil {
		t.Errorf("expected Basics.Email to be nil, got %v", resp.Agent.Basics.Email)
	}
	if resp.Agent.Campaign == nil || len(resp.Agent.Campaign.MessageExamples) != 3 {
		t.Fatalf("unexpected Campaign: %v", resp.Agent.Campaign)
	}
	if resp.Agent.Campaign.Interactions[0].InteractionType != "TRANSACTIONAL_UPDATES" {
		t.Errorf("unexpected Interactions: %v", resp.Agent.Campaign.Interactions)
	}
	if cs := resp.Agent.Campaign.ConsentSettings; cs == nil || cs.DoubleOptIn == nil || *cs.DoubleOptIn || cs.OptInMethods[0].MethodType != "WEBSITE" {
		t.Errorf("unexpected ConsentSettings: %v", cs)
	}
	if resp.Agent.Testing == nil || resp.Agent.Testing.TestURL != "https://example.com/test" {
		t.Errorf("unexpected Testing: %v", resp.Agent.Testing)
	}
	if len(resp.Agent.TestDevices) != 1 || resp.Agent.TestDevices[0].InviteStatus == nil || *resp.Agent.TestDevices[0].InviteStatus != "PENDING" {
		t.Errorf("unexpected TestDevices: %v", resp.Agent.TestDevices)
	}
	if len(resp.Devices) != 1 || resp.Devices[0].ID != "rcsd_1" {
		t.Errorf("unexpected Devices: %v", resp.Devices)
	}
}

func TestRCSRegistrationGet_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"brand": null, "agent": null, "devices": [], "stage": "draft", "usEligible": false}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.RCS.Registration.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Brand != nil || resp.Agent != nil {
		t.Errorf("expected nil Brand and Agent, got %v %v", resp.Brand, resp.Agent)
	}
	if len(resp.Devices) != 0 {
		t.Errorf("expected no devices, got %v", resp.Devices)
	}
	if resp.Stage != RcsCustomerStageDraft {
		t.Errorf("expected Stage to be 'draft', got '%s'", resp.Stage)
	}
	if resp.USEligible {
		t.Error("expected USEligible to be false")
	}
}

func TestRCSDossierGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/dossier" {
			t.Errorf("expected path '/rcs/dossier', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"brand": {
				"legalName": "Acme Holdings LLC",
				"displayName": "Acme",
				"ein": "123456789",
				"organizationType": "PRIVATE_PROFIT",
				"websiteUrl": "https://example.com",
				"address": {"line1": "1 Main St", "city": "Austin", "state": "TX", "postalCode": "78701", "countryCode": "US"},
				"contact": {"firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.com", "phoneNumber": "+15551234567"}
			},
			"usEligible": true,
			"source": "tendlc"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.RCS.Dossier.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Source != "tendlc" {
		t.Errorf("expected Source to be 'tendlc', got '%s'", resp.Source)
	}
	if !resp.USEligible {
		t.Error("expected USEligible to be true")
	}
	if resp.Brand.LegalName != "Acme Holdings LLC" || resp.Brand.EIN != "123456789" {
		t.Errorf("unexpected Brand: %+v", resp.Brand)
	}
	if resp.Brand.Address == nil || resp.Brand.Address.City != "Austin" {
		t.Errorf("unexpected Address: %v", resp.Brand.Address)
	}
	if resp.Brand.Contact == nil || resp.Brand.Contact.FirstName != "Ada" {
		t.Errorf("unexpected Contact: %v", resp.Brand.Contact)
	}
	if resp.Brand.LegalEntityType != "" {
		t.Errorf("expected LegalEntityType to be empty, got '%s'", resp.Brand.LegalEntityType)
	}
}

func TestRCSDossierGet_None(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"brand": {}, "usEligible": true, "source": "none"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Dossier.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Source != "none" || resp.Brand.Address != nil || resp.Brand.Contact != nil {
		t.Errorf("unexpected dossier: %+v", resp)
	}
}

func TestRCSBrandsCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/brands" {
			t.Errorf("expected path '/rcs/brands', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); !strings.HasPrefix(key, "sendly-go-retry-") {
			t.Errorf("expected an automatic Idempotency-Key, got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		if body["legalName"] != "Acme Holdings LLC" {
			t.Errorf("expected legalName to be 'Acme Holdings LLC', got '%v'", body["legalName"])
		}
		if body["legalEntityType"] != "LIMITED_LIABILITY_COMPANY" {
			t.Errorf("expected legalEntityType, got '%v'", body["legalEntityType"])
		}
		if body["ein"] != "12-3456789" {
			t.Errorf("expected ein to be '12-3456789', got '%v'", body["ein"])
		}
		if _, present := body["stockSymbol"]; present {
			t.Error("expected stockSymbol to be omitted")
		}
		address, _ := body["address"].(map[string]interface{})
		if address["line1"] != "1 Main St" || address["countryCode"] != "US" {
			t.Errorf("unexpected address: %v", body["address"])
		}
		if _, present := address["line2"]; present {
			t.Error("expected address.line2 to be omitted")
		}
		contact, _ := body["contact"].(map[string]interface{})
		if contact["email"] != "ada@example.com" || contact["phoneNumber"] != "+15551234567" {
			t.Errorf("unexpected contact: %v", body["contact"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"brand": ` + rcsBrandJSON + `}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.RCS.Brands.Create(ctx, &RcsBrandInput{
		DisplayName:      "Acme",
		LegalName:        "Acme Holdings LLC",
		LegalEntityType:  "LIMITED_LIABILITY_COMPANY",
		OrganizationType: "PRIVATE_PROFIT",
		WebsiteURL:       "https://example.com",
		EIN:              "12-3456789",
		Address: &RcsBrandAddress{
			Line1:       "1 Main St",
			City:        "Austin",
			State:       "TX",
			PostalCode:  "78701",
			CountryCode: "US",
		},
		Contact: &RcsBrandContact{
			FirstName:   "Ada",
			LastName:    "Lovelace",
			Email:       "ada@example.com",
			PhoneNumber: "+15551234567",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Brand.ID != "rcsb_1" {
		t.Errorf("expected ID to be 'rcsb_1', got '%s'", resp.Brand.ID)
	}
	if resp.Brand.ReviewStatus != RcsReviewStatusDraft {
		t.Errorf("expected ReviewStatus to be 'draft', got '%s'", resp.Brand.ReviewStatus)
	}
	if resp.Brand.CustomerStage != RcsCustomerStageDraft {
		t.Errorf("expected CustomerStage to be 'draft', got '%s'", resp.Brand.CustomerStage)
	}
	if resp.Brand.LegalEntityType != "LIMITED_LIABILITY_COMPANY" {
		t.Errorf("expected LegalEntityType, got '%s'", resp.Brand.LegalEntityType)
	}
	if resp.Brand.Contact.Title != "" {
		t.Errorf("expected Contact.Title to be empty, got '%s'", resp.Brand.Contact.Title)
	}
}

func TestRCSBrandsCreate_USOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error": "rcs_us_only", "message": "RCS registration is available to US businesses for now."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.RCS.Brands.Create(context.Background(), &RcsBrandInput{
		LegalName: "Acme Ltd",
		Address:   &RcsBrandAddress{CountryCode: "GB"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Code != RcsErrorCodeUSOnly {
		t.Errorf("expected Code to be 'rcs_us_only', got '%s'", validationErr.Code)
	}
}

func TestRCSBrandsCreate_RequiresRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with nil body")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.RCS.Brands.Create(context.Background(), nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestRCSBrandsUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/brands/rcsb_1" {
			t.Errorf("expected path '/rcs/brands/rcsb_1', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "brand-fix-1" {
			t.Errorf("expected Idempotency-Key to be 'brand-fix-1', got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		if len(body) != 2 {
			t.Errorf("expected only websiteUrl and contact, got %v", body)
		}
		if body["websiteUrl"] != "https://acme.example.com" {
			t.Errorf("expected websiteUrl, got '%v'", body["websiteUrl"])
		}
		contact, _ := body["contact"].(map[string]interface{})
		if len(contact) != 1 || contact["title"] != "CEO" {
			t.Errorf("expected contact to carry only title, got %v", body["contact"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"brand": ` + strings.Replace(rcsBrandJSON, `"title": null`, `"title": "CEO"`, 1) + `}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Brands.Update(context.Background(), "rcsb_1", &RcsBrandInput{
		WebsiteURL: "https://acme.example.com",
		Contact:    &RcsBrandContact{Title: "CEO"},
	}, WithIdempotencyKey("brand-fix-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Brand.Contact.Title != "CEO" {
		t.Errorf("expected Contact.Title to be 'CEO', got '%s'", resp.Brand.Contact.Title)
	}
}

func TestRCSBrandsUpdate_Locked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "rcs_field_locked", "message": "This registration is being reviewed; we will email you if changes are needed."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	_, err := client.RCS.Brands.Update(context.Background(), "rcsb_1", &RcsBrandInput{DisplayName: "Acme"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusConflict {
		t.Errorf("expected StatusCode to be 409, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != RcsErrorCodeFieldLocked {
		t.Errorf("expected Code to be 'rcs_field_locked', got '%s'", sendlyErr.Code)
	}
}

func TestRCSBrandsUpdate_RequiresID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty id")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.RCS.Brands.Update(context.Background(), "", &RcsBrandInput{})
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "brand ID is required") {
		t.Errorf("expected error to contain 'brand ID is required', got '%s'", err.Error())
	}
}

func TestRCSAgentsCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents" {
			t.Errorf("expected path '/rcs/agents', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key == "" {
			t.Error("expected an Idempotency-Key")
		}

		body := decodeRequestBody(t, r)
		if body["brandId"] != "rcsb_1" {
			t.Errorf("expected brandId to be 'rcsb_1', got '%v'", body["brandId"])
		}
		if body["displayName"] != "Acme Support" || body["useCase"] != "MULTI_USE" {
			t.Errorf("unexpected shorthands: %v %v", body["displayName"], body["useCase"])
		}
		basics, _ := body["basics"].(map[string]interface{})
		if basics["logoUrl"] != "https://example.com/logo.png" {
			t.Errorf("expected basics.logoUrl, got '%v'", basics["logoUrl"])
		}
		phone, _ := basics["phoneNumber"].(map[string]interface{})
		if phone["number"] != "+15551234567" || phone["label"] != "Support" {
			t.Errorf("unexpected basics.phoneNumber: %v", basics["phoneNumber"])
		}
		if _, present := basics["heroUrl"]; present {
			t.Error("expected basics.heroUrl to be omitted")
		}
		campaign, _ := body["campaign"].(map[string]interface{})
		interactions, _ := campaign["interactions"].([]interface{})
		if len(interactions) != 1 {
			t.Fatalf("expected 1 interaction, got %v", campaign["interactions"])
		}
		if interactions[0].(map[string]interface{})["interactionType"] != "TRANSACTIONAL_UPDATES" {
			t.Errorf("unexpected interaction: %v", interactions[0])
		}
		examples, _ := campaign["messageExamples"].([]interface{})
		if len(examples) != 3 {
			t.Errorf("expected 3 message examples, got %v", campaign["messageExamples"])
		}
		consent, _ := campaign["consentSettings"].(map[string]interface{})
		if consent["doubleOptIn"] != false {
			t.Errorf("expected consentSettings.doubleOptIn to be false, got %v", consent["doubleOptIn"])
		}
		methods, _ := consent["optInMethods"].([]interface{})
		if len(methods) != 1 || methods[0].(map[string]interface{})["methodType"] != "WEBSITE" {
			t.Errorf("unexpected optInMethods: %v", consent["optInMethods"])
		}
		testing, _ := body["testing"].(map[string]interface{})
		if testing["testUrl"] != "https://example.com/test" {
			t.Errorf("expected testing.testUrl, got '%v'", testing["testUrl"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"agent": ` + rcsAgentJSON + `}`))
	}))
	defer server.Close()

	doubleOptIn := false
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.Create(context.Background(), &CreateRcsAgentRequest{
		BrandID:     "rcsb_1",
		DisplayName: "Acme Support",
		UseCase:     "MULTI_USE",
		Basics: &RcsAgentBasics{
			Description: "Order updates and support",
			LogoURL:     "https://example.com/logo.png",
			BrandColor:  "#FF5500",
			PhoneNumber: &RcsAgentPhoneContact{Number: "+15551234567", Label: "Support"},
			Website:     &RcsAgentWebsiteContact{URL: "https://example.com", Label: "Site"},
		},
		Campaign: &RcsCampaign{
			AgentOverview:   "Sends order updates",
			Interactions:    []RcsInteraction{{InteractionType: "TRANSACTIONAL_UPDATES", Description: "Shipping"}},
			MessageExamples: []string{"Your order shipped", "Your order arrived", "Reply STOP to opt out"},
			ConsentSettings: &RcsConsentSettings{
				OptInMethods: []RcsOptInMethod{{MethodType: "WEBSITE", Description: "Checkout"}},
				DoubleOptIn:  &doubleOptIn,
			},
		},
		Testing: &RcsTesting{TestURL: "https://example.com/test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Agent.ID != "rcsa_1" {
		t.Errorf("expected ID to be 'rcsa_1', got '%s'", resp.Agent.ID)
	}
	if resp.Agent.Status != "draft" || resp.Agent.ReviewStatus != RcsReviewStatusDraft {
		t.Errorf("unexpected statuses: %s %s", resp.Agent.Status, resp.Agent.ReviewStatus)
	}
	if resp.Agent.UseCase == nil || *resp.Agent.UseCase != "MULTI_USE" {
		t.Errorf("expected UseCase to be 'MULTI_USE', got %v", resp.Agent.UseCase)
	}
	if resp.Agent.HostingRegion == nil || *resp.Agent.HostingRegion != "NORTH_AMERICA" {
		t.Errorf("expected HostingRegion, got %v", resp.Agent.HostingRegion)
	}
	if resp.Agent.Basics.Website == nil || resp.Agent.Basics.Website.URL != "https://example.com" {
		t.Errorf("unexpected Basics.Website: %v", resp.Agent.Basics.Website)
	}
	if resp.Stage != "" || len(resp.Devices) != 0 {
		t.Errorf("expected no top-level stage/devices on create, got '%s' %v", resp.Stage, resp.Devices)
	}
}

func TestRCSAgentsCreate_InvalidMediaURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"error": "rcs_invalid_content",
			"message": "Assets can't be uploaded over the API. Logo, hero, and call-to-action media must be public https:// URLs.",
			"errors": [{"path": "basics.logoUrl", "message": "Must be a public https:// URL"}]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.RCS.Agents.Create(context.Background(), &CreateRcsAgentRequest{
		BrandID: "rcsb_1",
		Basics:  &RcsAgentBasics{LogoURL: "http://example.com/logo.png"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Code != RcsErrorCodeInvalidContent {
		t.Errorf("expected Code to be 'rcs_invalid_content', got '%s'", validationErr.Code)
	}
	if len(validationErr.Errors) != 1 || validationErr.Errors[0].Path != "basics.logoUrl" {
		t.Fatalf("expected one field error at basics.logoUrl, got %v", validationErr.Errors)
	}
	if validationErr.Errors[0].Message != "Must be a public https:// URL" {
		t.Errorf("unexpected field message: %s", validationErr.Errors[0].Message)
	}
}

func TestRCSAgentsCreate_RequiresBrandID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request without brandId")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.RCS.Agents.Create(ctx, nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil request, got %T", err)
	}
	_, err := client.RCS.Agents.Create(ctx, &CreateRcsAgentRequest{DisplayName: "Acme"})
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "brandId is required") {
		t.Errorf("expected error to contain 'brandId is required', got '%s'", err.Error())
	}
}

func TestRCSAgentsGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents/rcsa_1" {
			t.Errorf("expected path '/rcs/agents/rcsa_1', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"agent": ` + strings.Replace(rcsAgentJSON, `"customerStage": "draft"`, `"customerStage": "testing"`, 1) + `,
			"devices": [{"id": "rcsd_1", "phoneNumber": "+15557654321", "label": null, "inviteStatus": null, "createdAt": "2026-07-02T00:00:00Z"}],
			"stage": "testing"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.Get(context.Background(), "rcsa_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Agent.ID != "rcsa_1" {
		t.Errorf("expected ID to be 'rcsa_1', got '%s'", resp.Agent.ID)
	}
	if resp.Agent.CustomerStage != RcsCustomerStageTesting || resp.Stage != RcsCustomerStageTesting {
		t.Errorf("expected stage 'testing', got '%s' / '%s'", resp.Agent.CustomerStage, resp.Stage)
	}
	if len(resp.Devices) != 1 || resp.Devices[0].Label != nil || resp.Devices[0].InviteStatus != nil {
		t.Errorf("unexpected Devices: %v", resp.Devices)
	}
}

func TestRCSAgentsGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/rcs/agents/rcsa%2Fmissing" {
			t.Errorf("expected escaped path, got '%s'", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "rcs_not_found", "message": "Agent not found"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.RCS.Agents.Get(context.Background(), "rcsa/missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	notFound, ok := err.(*NotFoundError)
	if !ok {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
	if notFound.Code != RcsErrorCodeNotFound {
		t.Errorf("expected Code to be 'rcs_not_found', got '%s'", notFound.Code)
	}
}

func TestRCSAgentsUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents/rcsa_1" {
			t.Errorf("expected path '/rcs/agents/rcsa_1', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "agent-campaign-2" {
			t.Errorf("expected Idempotency-Key to be 'agent-campaign-2', got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		if len(body) != 1 {
			t.Errorf("expected only campaign in body, got %v", body)
		}
		campaign, _ := body["campaign"].(map[string]interface{})
		if campaign["companyOverview"] != "Family bakery in Austin" {
			t.Errorf("unexpected campaign: %v", campaign)
		}
		if _, present := campaign["interactions"]; present {
			t.Error("expected campaign.interactions to be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent": ` + rcsAgentJSON + `}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.Update(context.Background(), "rcsa_1", &UpdateRcsAgentRequest{
		Campaign: &RcsCampaign{CompanyOverview: "Family bakery in Austin"},
	}, WithIdempotencyKey("agent-campaign-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Agent.ID != "rcsa_1" {
		t.Errorf("expected ID to be 'rcsa_1', got '%s'", resp.Agent.ID)
	}
}

func TestRCSAgentsUpdate_RequiresIDAndRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.RCS.Agents.Update(ctx, "", &UpdateRcsAgentRequest{}); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %T", err)
	}
	if _, err := client.RCS.Agents.Update(ctx, "rcsa_1", nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil request, got %T", err)
	}
	if _, err := client.RCS.Agents.Get(ctx, ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id on Get, got %T", err)
	}
}

func TestRCSAgentsSetTestDevices_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents/rcsa_1/test-devices" {
			t.Errorf("expected path '/rcs/agents/rcsa_1/test-devices', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "devices-v3" {
			t.Errorf("expected Idempotency-Key to be 'devices-v3', got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		devices, _ := body["devices"].([]interface{})
		if len(devices) != 2 {
			t.Fatalf("expected 2 devices, got %v", body["devices"])
		}
		first := devices[0].(map[string]interface{})
		if first["phoneNumber"] != "+15557654321" || first["label"] != "Ada" {
			t.Errorf("unexpected first device: %v", first)
		}
		second := devices[1].(map[string]interface{})
		if _, present := second["label"]; present {
			t.Error("expected empty label to be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"devices": [
			{"id": "rcsd_1", "phoneNumber": "+15557654321", "label": "Ada", "inviteStatus": "PENDING", "createdAt": "2026-07-02T00:00:00Z"},
			{"id": "rcsd_2", "phoneNumber": "+15550001111", "label": null, "inviteStatus": null, "createdAt": "2026-07-03T00:00:00Z"}
		]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.SetTestDevices(context.Background(), "rcsa_1", []RcsTestDeviceInput{
		{PhoneNumber: "+15557654321", Label: "Ada"},
		{PhoneNumber: "(555) 000-1111"},
	}, WithIdempotencyKey("devices-v3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(resp.Devices))
	}
	if resp.Devices[0].Label == nil || *resp.Devices[0].Label != "Ada" {
		t.Errorf("expected first Label to be 'Ada', got %v", resp.Devices[0].Label)
	}
	if resp.Devices[1].Label != nil || resp.Devices[1].InviteStatus != nil {
		t.Errorf("expected second device to have nil Label and InviteStatus, got %v", resp.Devices[1])
	}
}

func TestRCSAgentsSetTestDevices_EmptyListSendsArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		devices, ok := body["devices"].([]interface{})
		if !ok || len(devices) != 0 {
			t.Errorf("expected devices to be an empty array, got %v", body["devices"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"devices": []}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.SetTestDevices(context.Background(), "rcsa_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Devices) != 0 {
		t.Errorf("expected no devices, got %v", resp.Devices)
	}
}

func TestRCSAgentsSetTestDevices_LocalValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.RCS.Agents.SetTestDevices(ctx, "", nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %T", err)
	}
	_, err := client.RCS.Agents.SetTestDevices(ctx, "rcsa_1", []RcsTestDeviceInput{{Label: "no number"}})
	if !IsValidationError(err) || !strings.Contains(err.Error(), "devices[0].phoneNumber is required") {
		t.Errorf("expected phoneNumber validation error, got %v", err)
	}
	tooMany := make([]RcsTestDeviceInput, 21)
	for i := range tooMany {
		tooMany[i].PhoneNumber = "+15550000000"
	}
	_, err = client.RCS.Agents.SetTestDevices(ctx, "rcsa_1", tooMany)
	if !IsValidationError(err) || !strings.Contains(err.Error(), "up to 20 test devices") {
		t.Errorf("expected 20-device validation error, got %v", err)
	}
}

func TestRCSAgentsSubmit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents/rcsa_1/submit" {
			t.Errorf("expected path '/rcs/agents/rcsa_1/submit', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "submit-rcsa_1" {
			t.Errorf("expected Idempotency-Key to be 'submit-rcsa_1', got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		if len(body) != 0 {
			t.Errorf("expected empty JSON body, got %v", body)
		}

		agent := strings.Replace(rcsAgentJSON, `"reviewStatus": "draft"`, `"reviewStatus": "awaiting_review"`, 1)
		agent = strings.Replace(agent, `"customerStage": "draft"`, `"customerStage": "in_review"`, 1)
		agent = strings.Replace(agent, `"submittedForReviewAt": null`, `"submittedForReviewAt": "2026-07-03T00:00:00Z"`, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent": ` + agent + `, "stage": "in_review"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.Submit(context.Background(), "rcsa_1", WithIdempotencyKey("submit-rcsa_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Agent.ReviewStatus != RcsReviewStatusAwaitingReview {
		t.Errorf("expected ReviewStatus 'awaiting_review', got '%s'", resp.Agent.ReviewStatus)
	}
	if resp.Stage != RcsCustomerStageInReview || resp.Agent.CustomerStage != RcsCustomerStageInReview {
		t.Errorf("expected stage 'in_review', got '%s' / '%s'", resp.Stage, resp.Agent.CustomerStage)
	}
	if resp.Agent.SubmittedForReviewAt == nil {
		t.Error("expected SubmittedForReviewAt to be set")
	}
}

func TestRCSAgentsSubmit_AutoIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key := r.Header.Get("Idempotency-Key"); !strings.HasPrefix(key, "sendly-go-retry-") {
			t.Errorf("expected an automatic Idempotency-Key, got '%s'", key)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent": ` + rcsAgentJSON + `, "stage": "in_review"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.RCS.Agents.Submit(context.Background(), "rcsa_1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := client.RCS.Agents.Submit(context.Background(), ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %T", err)
	}
}

func TestRCSAgentsSubmit_Incomplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"error": "rcs_invalid_content",
			"message": "Finish the brand and agent basics before submitting.",
			"errors": [
				{"path": "brand.ein", "message": "Enter a 9-digit EIN"},
				{"path": "agent.logoUrl", "message": "Must be a public https:// URL"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.RCS.Agents.Submit(context.Background(), "rcsa_1")
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Code != RcsErrorCodeInvalidContent || len(validationErr.Errors) != 2 {
		t.Fatalf("unexpected error: code=%s errors=%v", validationErr.Code, validationErr.Errors)
	}
	if validationErr.Errors[0].Path != "brand.ein" || validationErr.Errors[1].Path != "agent.logoUrl" {
		t.Errorf("unexpected field paths: %v", validationErr.Errors)
	}
}

func TestRCSAgentsRequestLaunch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents/rcsa_1/request-launch" {
			t.Errorf("expected path '/rcs/agents/rcsa_1/request-launch', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key == "" {
			t.Error("expected an Idempotency-Key")
		}

		body := decodeRequestBody(t, r)
		if body["testUrl"] != "https://example.com/test" {
			t.Errorf("expected testUrl, got '%v'", body["testUrl"])
		}
		if body["testingAdditionalInformation"] != "Reply START to begin" {
			t.Errorf("expected testingAdditionalInformation, got '%v'", body["testingAdditionalInformation"])
		}

		agent := strings.Replace(rcsAgentJSON, `"reviewStatus": "draft"`, `"reviewStatus": "launch_requested"`, 1)
		agent = strings.Replace(agent, `"customerStage": "draft"`, `"customerStage": "launch_review"`, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent": ` + agent + `, "stage": "launch_review"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.RCS.Agents.RequestLaunch(context.Background(), "rcsa_1", &RcsLaunchRequest{
		TestURL:                      "https://example.com/test",
		TestingAdditionalInformation: "Reply START to begin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Agent.ReviewStatus != RcsReviewStatusLaunchRequested {
		t.Errorf("expected ReviewStatus 'launch_requested', got '%s'", resp.Agent.ReviewStatus)
	}
	if resp.Stage != RcsCustomerStageLaunchReview {
		t.Errorf("expected Stage 'launch_review', got '%s'", resp.Stage)
	}
}

func TestRCSAgentsRequestLaunch_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 0 {
			t.Errorf("expected empty JSON body, got %v", body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent": ` + rcsAgentJSON + `, "stage": "launch_review"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.RCS.Agents.RequestLaunch(context.Background(), "rcsa_1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := client.RCS.Agents.RequestLaunch(context.Background(), "", nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %T", err)
	}
}

func TestRCSAgentsRequestLaunch_NotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "rcs_launch_not_ready", "message": "This agent isn't ready to launch yet. Finish testing on an invited device first."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	_, err := client.RCS.Agents.RequestLaunch(context.Background(), "rcsa_1", nil)
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusConflict || sendlyErr.Code != RcsErrorCodeLaunchNotReady {
		t.Errorf("expected 409 rcs_launch_not_ready, got %d %s", sendlyErr.StatusCode, sendlyErr.Code)
	}
}

func TestRCSRegistration_NotEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "rcs_not_enabled", "message": "RCS registration isn't enabled for this account yet."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	calls := map[string]func() error{
		"registration.get": func() error { _, err := client.RCS.Registration.Get(ctx); return err },
		"dossier.get":      func() error { _, err := client.RCS.Dossier.Get(ctx); return err },
		"brands.create": func() error {
			_, err := client.RCS.Brands.Create(ctx, &RcsBrandInput{LegalName: "Acme"})
			return err
		},
		"brands.update": func() error {
			_, err := client.RCS.Brands.Update(ctx, "rcsb_1", &RcsBrandInput{LegalName: "Acme"})
			return err
		},
		"agents.create": func() error {
			_, err := client.RCS.Agents.Create(ctx, &CreateRcsAgentRequest{BrandID: "rcsb_1"})
			return err
		},
		"agents.get": func() error { _, err := client.RCS.Agents.Get(ctx, "rcsa_1"); return err },
		"agents.update": func() error {
			_, err := client.RCS.Agents.Update(ctx, "rcsa_1", &UpdateRcsAgentRequest{DisplayName: "Acme"})
			return err
		},
		"agents.setTestDevices": func() error {
			_, err := client.RCS.Agents.SetTestDevices(ctx, "rcsa_1", nil)
			return err
		},
		"agents.submit":        func() error { _, err := client.RCS.Agents.Submit(ctx, "rcsa_1"); return err },
		"agents.requestLaunch": func() error { _, err := client.RCS.Agents.RequestLaunch(ctx, "rcsa_1", nil); return err },
	}

	for name, call := range calls {
		err := call()
		if err == nil {
			t.Errorf("%s: expected error, got nil", name)
			continue
		}
		notFound, ok := err.(*NotFoundError)
		if !ok {
			t.Errorf("%s: expected NotFoundError, got %T", name, err)
			continue
		}
		if notFound.Code != RcsErrorCodeNotEnabled {
			t.Errorf("%s: expected Code to be 'rcs_not_enabled', got '%s'", name, notFound.Code)
		}
		if notFound.Message != "RCS registration isn't enabled for this account yet." {
			t.Errorf("%s: unexpected Message '%s'", name, notFound.Message)
		}
	}
}
