package sendly

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const shortCodeLeaseJSON = `{"monthlyCents":100000,"minimumTermMonths":3,"state":"not_started","billingEnabled":true,"startedAt":null,"nextChargeAt":null,"paidThrough":null,"termEndsAt":null,"endsAt":null,"cancelRequestedAt":null,"pastDue":null,"dueMonths":0,"dueCents":0}`

const shortCodeViewJSON = `{
  "enabled": true,
  "application": {
    "id": "sc_1", "shortCode": null, "countryCode": "US", "status": "requested", "reviewStatus": "awaiting_review",
    "reviewNote": null, "orderType": "new", "codeType": "random", "contentProviderSameAsBrand": true,
    "sampleMessages": ["Acme: your order shipped. Reply STOP to opt out."],
    "useCase": "Order and delivery alerts", "optInFlow": "Customers text JOIN to the code after checkout.",
    "optInConfirmation": null, "helpResponse": null, "stopConfirmation": null,
    "messageFrequency": "Up to 4 messages per order", "campaignKeyword": "JOIN",
    "expectedMonthlyVolume": "50000", "expectedDailyVolume": "2000",
    "privacyPolicyUrl": "https://acme.example/privacy", "termsUrl": "https://acme.example/terms",
    "requestedDigits": null, "losingProvider": null,
    "brandContactName": "Ada Lovelace", "brandContactEmail": "ada@acme.example", "brandContactPhone": "+15125550100",
    "contentProviderLegalName": null, "contentProviderEin": null, "contentProviderContactName": null,
    "contentProviderContactEmail": null, "contentProviderContactPhone": null,
    "brandRegistrationStatus": "registered", "contentProviderRegistrationStatus": "not_started",
    "submittedAt": "2026-09-01T10:00:00.000Z", "reviewedAt": null, "filedAt": null, "activatedAt": null,
    "registryRevettingDueAt": null, "createdAt": "2026-08-30T10:00:00.000Z", "updatedAt": "2026-09-01T10:00:00.000Z"
  },
  "prefill": {"source": "tendlc", "legalName": "Acme Inc", "address": {"city": "Austin"}, "contact": {"email": "ada@acme.example"}},
  "editable": false,
  "lockMessage": "Sendly is reviewing this application.",
  "canStartNewApplication": false,
  "generatedDefaults": null,
  "requiredDocuments": ["order_brief", "loa"],
  "missingDocuments": ["loa"],
  "documents": [{"kind": "order_brief", "templateReady": true, "signedAt": "2026-09-02T10:00:00.000Z", "signedByName": "Ada Lovelace", "awaitingSignature": false}],
  "quote": {"codeType": "random", "monthlyUsd": 1000, "setupUsd": 999, "currency": "USD", "autoBilled": false, "setupCents": 99900, "monthlyCents": 100000, "minimumTermMonths": 3, "setupChargedAt": "submit", "leaseStartsAt": "go_live"},
  "billing": {
    "setupFee": {"amountCents": 99900, "status": "paid", "chargedAt": "submit", "paidAt": "2026-09-01T10:00:00.000Z", "refundedCents": 0, "refundedAt": null},
    "lease": {"monthlyCents":100000,"minimumTermMonths":3,"startsAt":"go_live","state":"not_started","billingEnabled":true,"startedAt":null,"nextChargeAt":null,"paidThrough":null,"termEndsAt":null,"endsAt":null,"cancelRequestedAt":null,"pastDue":null,"dueMonths":0,"dueCents":0},
    "terms": {"version": "2026-09", "acceptedVersion": "2026-09", "acceptedAt": "2026-09-01T10:00:00.000Z"},
    "refundPolicy": "full_refund_before_filing",
    "exempt": null,
    "charges": [{"id": "scc_1", "kind": "setup", "amountCents": 99900, "status": "paid", "paidAt": "2026-09-01T10:00:00.000Z", "refundedCents": 0, "refundedAt": null, "periodStart": null, "periodEnd": null, "createdAt": "2026-09-01T10:00:00.000Z"}]
  },
  "verification": {"verified": true, "state": "verified", "href": "/verify"},
  "carriers": {"approved": 0, "total": 4, "overall": "not_submitted", "byCarrier": [{"carrier": "ATT", "label": "AT&T", "status": "not_submitted", "updatedAt": null}]}
  EXTRA
}`

func shortCodeView(extra string) string {
	return strings.Replace(shortCodeViewJSON, "EXTRA", extra, 1)
}

func TestShortCodesList_DecodesTheCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/short_codes" {
			t.Errorf("expected GET /short_codes, got %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"shortCodes":[{"id":"sc_1","shortCode":"12345","countryCode":"US","status":"active","reviewStatus":"filed","useCase":"Order alerts","createdAt":"2026-08-30T10:00:00.000Z","billing":{"setupFee":{"status":"paid","amountCents":99900},"lease":` + shortCodeLeaseJSON + `}},{"id":"sc_0","shortCode":null,"countryCode":"US","status":"cancelled","reviewStatus":"rejected","useCase":"Old","createdAt":"2026-07-01T10:00:00.000Z","billing":{"setupFee":{"status":"refunded","amountCents":99900},"lease":` + shortCodeLeaseJSON + `}}]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	list, err := client.ShortCodes.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.ShortCodes) != 2 {
		t.Fatalf("expected 2 short codes, got %d", len(list.ShortCodes))
	}
	first := list.ShortCodes[0]
	if first.ShortCode == nil || *first.ShortCode != "12345" || first.Status != ShortCodeStatusActive || first.ReviewStatus != ShortCodeReviewStatusFiled {
		t.Errorf("unexpected first code: %+v", first)
	}
	if first.Billing.SetupFee.Status != "paid" || first.Billing.SetupFee.AmountCents != 99900 || first.Billing.Lease.MonthlyCents != 100000 || first.Billing.Lease.PastDue != nil {
		t.Errorf("unexpected billing: %+v", first.Billing)
	}
	if list.ShortCodes[1].ShortCode != nil {
		t.Errorf("expected nil ShortCode on a draft, got %v", *list.ShortCodes[1].ShortCode)
	}
}

func TestShortCodeApplicationGet_DecodesTheView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/short_codes/application" {
			t.Errorf("expected GET /short_codes/application, got %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(shortCodeView("")))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	view, err := client.ShortCodes.Application.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	app := view.Application
	if app.ID == nil || *app.ID != "sc_1" || app.ShortCode != nil || app.ReviewStatus != ShortCodeReviewStatusAwaitingReview {
		t.Errorf("unexpected application: %+v", app)
	}
	if app.UseCase == nil || *app.UseCase != "Order and delivery alerts" || app.HelpResponse != nil {
		t.Errorf("unexpected answers: %+v", app)
	}
	if view.Editable || view.LockMessage == nil || view.GeneratedDefaults != nil {
		t.Errorf("unexpected lock state: %+v", view)
	}
	if view.Prefill.Source != "tendlc" || view.Prefill.Address == nil || view.Prefill.Address.City != "Austin" {
		t.Errorf("unexpected prefill: %+v", view.Prefill)
	}
	if len(view.MissingDocuments) != 1 || view.MissingDocuments[0] != "loa" || len(view.Documents) != 1 || view.Documents[0].SignedByName == nil {
		t.Errorf("unexpected documents: %+v %+v", view.MissingDocuments, view.Documents)
	}
	if view.Quote.SetupCents != 99900 || view.Quote.LeaseStartsAt != "go_live" {
		t.Errorf("unexpected quote: %+v", view.Quote)
	}
	if view.Billing.SetupFee.Status != "paid" || view.Billing.Lease.StartsAt != "go_live" || view.Billing.Exempt != nil || len(view.Billing.Charges) != 1 || view.Billing.Charges[0].Kind != "setup" {
		t.Errorf("unexpected billing: %+v", view.Billing)
	}
	if view.Carriers.Total != 4 || view.Carriers.ByCarrier[0].Carrier != "ATT" || !view.Verification.Verified {
		t.Errorf("unexpected carriers or verification: %+v %+v", view.Carriers, view.Verification)
	}
	if view.Payment != nil || view.Submitted {
		t.Errorf("expected no submit fields on a get")
	}
}

func TestShortCodeApplicationUpdate_PutsTheAnswersWithTheCallerKey(t *testing.T) {
	var raw string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/short_codes/application" {
			t.Errorf("expected PUT /short_codes/application, got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "save-1" {
			t.Errorf("expected caller idempotency key, got %q", got)
		}
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		w.Write([]byte(shortCodeView(`, "ignoredFields": ["status"]`)))
	}))
	defer server.Close()

	useCase := "Delivery alerts for Acme orders"
	clear := ""
	sameAsBrand := false
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	view, err := client.ShortCodes.Application.Update(context.Background(), &ShortCodeApplicationInput{
		UseCase:                    &useCase,
		HelpResponse:               &clear,
		CodeType:                   "vanity",
		ContentProviderSameAsBrand: &sameAsBrand,
		SampleMessages:             []string{"Acme: your order shipped."},
	}, WithIdempotencyKey("save-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("bad body: %v", err)
	}
	if body["useCase"] != useCase || body["helpResponse"] != "" || body["codeType"] != "vanity" || body["contentProviderSameAsBrand"] != false {
		t.Errorf("unexpected body: %s", raw)
	}
	if _, ok := body["optInFlow"]; ok {
		t.Errorf("expected unset fields to be left out: %s", raw)
	}
	if len(view.IgnoredFields) != 1 || view.IgnoredFields[0] != "status" {
		t.Errorf("expected ignoredFields [status], got %v", view.IgnoredFields)
	}
}

func TestShortCodeApplicationCheck_PostsToPreflight(t *testing.T) {
	var raw string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/short_codes/application/preflight" {
			t.Errorf("expected POST /short_codes/application/preflight, got %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		w.Write([]byte(`{"ok":false,"issues":[{"path":"helpResponse","message":"Add a HELP reply."}],"requiredDocuments":["order_brief"],"missingDocuments":["order_brief"],"documents":[]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	result, err := client.ShortCodes.Application.Check(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw != "{}" {
		t.Errorf("expected an empty body, got %s", raw)
	}
	if result.OK || len(result.Issues) != 1 || result.Issues[0].Path != "helpResponse" || result.MissingDocuments[0] != "order_brief" {
		t.Errorf("unexpected preflight: %+v", result)
	}
}

func TestShortCodeApplicationSubmit_SendsAcceptTermsAndReadsThePayment(t *testing.T) {
	var raw string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/short_codes/application/submit" {
			t.Errorf("expected POST /short_codes/application/submit, got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "submit-1" {
			t.Errorf("expected caller idempotency key, got %q", got)
		}
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		w.Write([]byte(shortCodeView(`, "submitted": true, "alreadySubmitted": false, "notified": true, "payment": {"status": "paid", "charged": true}`)))
	}))
	defer server.Close()

	keyword := "JOIN"
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	view, err := client.ShortCodes.Application.Submit(context.Background(), &ShortCodeSubmitInput{
		ShortCodeApplicationInput: ShortCodeApplicationInput{CampaignKeyword: &keyword},
		AcceptTerms:               true,
	}, WithIdempotencyKey("submit-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("bad body: %v", err)
	}
	if body["acceptTerms"] != true || body["campaignKeyword"] != "JOIN" || len(body) != 2 {
		t.Errorf("unexpected body: %s", raw)
	}
	if !view.Submitted || view.AlreadySubmitted || !view.Notified || view.Payment == nil || !view.Payment.Charged || view.Payment.Status != "paid" {
		t.Errorf("unexpected submit answer: %+v", view)
	}
}

func TestShortCodeApplicationSubmit_ReturnsTheVerificationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"verification_required","message":"Verify your business first."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.ShortCodes.Application.Submit(context.Background(), &ShortCodeSubmitInput{AcceptTerms: true})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Verify your business first.") {
		t.Errorf("unexpected error: %v", err)
	}
}
