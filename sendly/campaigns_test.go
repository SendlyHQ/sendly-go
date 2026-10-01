package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const wireCampaignRow = `"id":"camp_1","userId":"usr_1","organizationId":null,"name":"Fall","status":"completed","messageText":"Hello","fromSender":null,"targetType":"contact_list","targetListId":"lst_1","manualRecipients":null,"excludeOptedOut":true,"sendNow":false,"scheduledAt":null,"timezone":"America/New_York","batchId":"batch_1","totalRecipients":5,"estimatedCredits":10,"sentCount":5,"deliveredCount":4,"failedCount":1,"creditsUsed":10,"creditsRefunded":0,"createdAt":"2026-09-25T09:00:00.000Z","updatedAt":"2026-09-25T10:01:00.000Z","sentAt":"2026-09-25T10:00:00.000Z","completedAt":"2026-09-25T10:01:00.000Z"`

const wireCampaignTwins = `,"text":"Hello","contact_list_ids":["lst_1"],"created_at":"2026-09-25T09:00:00.000Z","updated_at":"2026-09-25T10:01:00.000Z"`

func campaignServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not_found","message":"Campaign not found"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
}

func TestCampaignsGet_DecodesTheCampaignRow(t *testing.T) {
	shapes := map[string]string{
		"row only":      "{" + wireCampaignRow + "}",
		"row and twins": "{" + wireCampaignRow + wireCampaignTwins + "}",
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			server := campaignServer(t, map[string]string{"GET /campaigns/camp_1": body})
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))

			campaign, err := client.Campaigns.Get(context.Background(), "camp_1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if campaign.ID != "camp_1" || campaign.Name != "Fall" || campaign.Status != "completed" {
				t.Errorf("unexpected identity: %+v", campaign)
			}
			if campaign.Text != "Hello" {
				t.Errorf("expected Text 'Hello', got %q", campaign.Text)
			}
			if len(campaign.ContactListIDs) != 1 || campaign.ContactListIDs[0] != "lst_1" {
				t.Errorf("expected ContactListIDs [lst_1], got %v", campaign.ContactListIDs)
			}
			if campaign.RecipientCount != 5 || campaign.SentCount != 5 || campaign.DeliveredCount != 4 || campaign.FailedCount != 1 {
				t.Errorf("expected counts 5/5/4/1, got %d/%d/%d/%d", campaign.RecipientCount, campaign.SentCount, campaign.DeliveredCount, campaign.FailedCount)
			}
			if campaign.EstimatedCredits == nil || *campaign.EstimatedCredits != 10 || campaign.CreditsUsed == nil || *campaign.CreditsUsed != 10 {
				t.Errorf("expected EstimatedCredits and CreditsUsed 10, got %v and %v", campaign.EstimatedCredits, campaign.CreditsUsed)
			}
			if campaign.StartedAt == nil || *campaign.StartedAt != "2026-09-25T10:00:00.000Z" {
				t.Errorf("expected StartedAt from sentAt, got %v", campaign.StartedAt)
			}
			if campaign.CompletedAt == nil || *campaign.CompletedAt != "2026-09-25T10:01:00.000Z" {
				t.Errorf("expected CompletedAt, got %v", campaign.CompletedAt)
			}
			if campaign.ScheduledAt != nil {
				t.Errorf("expected no ScheduledAt, got %v", *campaign.ScheduledAt)
			}
			if campaign.CreatedAt != "2026-09-25T09:00:00.000Z" || campaign.UpdatedAt != "2026-09-25T10:01:00.000Z" {
				t.Errorf("expected CreatedAt and UpdatedAt, got %q and %q", campaign.CreatedAt, campaign.UpdatedAt)
			}
			if campaign.BatchID != "batch_1" || campaign.TargetType != "contact_list" || campaign.FromSender != nil {
				t.Errorf("expected BatchID batch_1, TargetType contact_list and no FromSender, got %+v", campaign)
			}
			if campaign.Timezone == nil || *campaign.Timezone != "America/New_York" {
				t.Errorf("expected Timezone America/New_York, got %v", campaign.Timezone)
			}
		})
	}
}

func TestCampaignsList_DecodesTheCampaignRows(t *testing.T) {
	server := campaignServer(t, map[string]string{
		"GET /campaigns": `{"campaigns":[{` + wireCampaignRow + wireCampaignTwins + `,"targetList":{"id":"lst_1","name":"Customers"}}],"total":1,"limit":50,"offset":0}`,
	})
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	list, err := client.Campaigns.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Total != 1 || len(list.Campaigns) != 1 {
		t.Fatalf("expected one campaign, got %+v", list)
	}
	if list.Campaigns[0].SentCount != 5 || list.Campaigns[0].DeliveredCount != 4 {
		t.Errorf("expected counts 5 and 4, got %+v", list.Campaigns[0])
	}
}

func TestCampaign_RoundTrip(t *testing.T) {
	var original Campaign
	if err := json.Unmarshal([]byte("{"+wireCampaignRow+wireCampaignTwins+"}"), &original); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded Campaign
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestCampaignsSchedule_SendsScheduledAt(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/campaigns/camp_1/schedule" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body = nil
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if scheduledAt, _ := body["scheduledAt"].(string); scheduledAt == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_request","message":"scheduledAt is required"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"camp_1","name":"Fall","status":"scheduled","messageText":"Hello","targetListId":"lst_1","scheduledAt":"2026-10-01T15:00:00.000Z","timezone":"America/Chicago","totalRecipients":0,"sentCount":0,"deliveredCount":0,"failedCount":0,"estimatedCredits":0,"creditsUsed":0,"creditsRefunded":0,"createdAt":"2026-09-25T09:00:00.000Z","updatedAt":"2026-09-25T09:05:00.000Z","sentAt":null,"completedAt":null,"text":"Hello","contact_list_ids":["lst_1"],"created_at":"2026-09-25T09:00:00.000Z","updated_at":"2026-09-25T09:05:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	campaign, err := client.Campaigns.Schedule(context.Background(), "camp_1", &ScheduleCampaignRequest{
		ScheduledAt: "2026-10-01T15:00:00Z",
		Timezone:    "America/Chicago",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["scheduledAt"] != "2026-10-01T15:00:00Z" || body["timezone"] != "America/Chicago" {
		t.Errorf("expected scheduledAt and timezone in the body, got %v", body)
	}
	if _, ok := body["scheduled_at"]; ok {
		t.Errorf("expected no scheduled_at key, got %v", body)
	}
	if campaign.Status != "scheduled" || campaign.ScheduledAt == nil || *campaign.ScheduledAt != "2026-10-01T15:00:00.000Z" {
		t.Errorf("expected a scheduled campaign, got %+v", campaign)
	}
}

func TestCampaignsList_FiltersByCompleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "completed" {
			t.Errorf("expected status=completed, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"campaigns":[{` + wireCampaignRow + wireCampaignTwins + `}],"total":1,"limit":50,"offset":0}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	list, err := client.Campaigns.List(context.Background(), &ListCampaignsRequest{Status: CampaignStatusCompleted})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Campaigns) != 1 || list.Campaigns[0].Status != string(CampaignStatusCompleted) {
		t.Errorf("expected a completed campaign, got %+v", list.Campaigns)
	}
}

const wireCampaignSendResult = `{"batchId":"batch_1","status":"completed","total":3,"sent":3,"failed":0,"retrying":0,"optedOutSkipped":0,"invalidSkipped":0,"creditsUsed":6,"creditsRefunded":0,"messages":[{"index":0,"id":"msg_1","to":"+15551230001","status":"sent"}]}`

func TestCampaignsSend_ReturnsTheSentCampaign(t *testing.T) {
	server := campaignServer(t, map[string]string{
		"POST /campaigns/camp_1/send": wireCampaignSendResult,
		"GET /campaigns/camp_1":       "{" + wireCampaignRow + wireCampaignTwins + "}",
	})
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	campaign, err := client.Campaigns.Send(context.Background(), "camp_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if campaign.ID != "camp_1" || campaign.Name != "Fall" {
		t.Errorf("expected campaign camp_1 'Fall', got ID %q name %q", campaign.ID, campaign.Name)
	}
	if campaign.Status != string(CampaignStatusCompleted) || campaign.BatchID != "batch_1" {
		t.Errorf("expected a completed campaign in batch_1, got status %q batch %q", campaign.Status, campaign.BatchID)
	}
}

func TestCampaignsSend_FallsBackWhenTheCampaignCannotBeRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(wireCampaignSendResult))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"insufficient_scope","message":"This API key lacks the campaigns:read scope"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	campaign, err := client.Campaigns.Send(context.Background(), "camp_1")
	if err != nil {
		t.Fatalf("expected the send to succeed, got %v", err)
	}
	if campaign.ID != "camp_1" || campaign.BatchID != "batch_1" || campaign.Status != string(CampaignStatusCompleted) {
		t.Errorf("unexpected campaign: %+v", campaign)
	}
	if campaign.RecipientCount != 3 || campaign.SentCount != 3 || campaign.FailedCount != 0 {
		t.Errorf("expected counts 3/3/0, got %+v", campaign)
	}
	if campaign.CreditsUsed == nil || *campaign.CreditsUsed != 6 {
		t.Errorf("expected CreditsUsed 6, got %v", campaign.CreditsUsed)
	}
}

const wireCampaignSendAllFailed = `{"batchId":"batch_2","status":"failed","total":2,"sent":0,"failed":2,"retrying":0,"optedOutSkipped":0,"invalidSkipped":0,"creditsUsed":0,"creditsRefunded":4,"messages":[{"index":0,"id":"msg_1","to":"+15551230001","status":"failed","error":"Carrier rejected"},{"index":1,"id":"msg_2","to":"+15551230002","status":"failed","error":"Carrier rejected"}]}`

const wireCampaignRowAllFailed = `{"id":"camp_2","userId":"usr_1","organizationId":null,"name":"Winter","status":"completed","messageText":"Hello","fromSender":null,"targetType":"contact_list","targetListId":"lst_1","manualRecipients":null,"excludeOptedOut":true,"sendNow":false,"scheduledAt":null,"timezone":"America/New_York","batchId":"batch_2","totalRecipients":2,"estimatedCredits":0,"sentCount":0,"deliveredCount":0,"failedCount":2,"creditsUsed":0,"creditsRefunded":0,"createdAt":"2026-09-25T09:00:00.000Z","updatedAt":"2026-09-25T10:01:00.000Z","sentAt":"2026-09-25T10:00:00.000Z","completedAt":"2026-09-25T10:01:00.000Z"}`

func TestCampaignsSend_StatusIsWhatTheSendReported(t *testing.T) {
	t.Run("campaign read back", func(t *testing.T) {
		server := campaignServer(t, map[string]string{
			"POST /campaigns/camp_2/send": wireCampaignSendAllFailed,
			"GET /campaigns/camp_2":       wireCampaignRowAllFailed,
		})
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		campaign, err := client.Campaigns.Send(context.Background(), "camp_2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if campaign.Status != string(CampaignStatusFailed) {
			t.Errorf("expected Status %q, the batch status the send reported, got %q", CampaignStatusFailed, campaign.Status)
		}
		if campaign.ID != "camp_2" || campaign.Name != "Winter" || campaign.BatchID != "batch_2" || campaign.FailedCount != 2 {
			t.Errorf("expected the campaign read back, got %+v", campaign)
		}
	})

	t.Run("campaign not readable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(wireCampaignSendAllFailed))
				return
			}
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"insufficient_scope","message":"This API key lacks the campaigns:read scope"}`))
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		campaign, err := client.Campaigns.Send(context.Background(), "camp_2")
		if err != nil {
			t.Fatalf("expected the send to succeed, got %v", err)
		}
		if campaign.Status != string(CampaignStatusFailed) {
			t.Errorf("expected Status %q, the batch status the send reported, got %q", CampaignStatusFailed, campaign.Status)
		}
		if campaign.ID != "camp_2" || campaign.BatchID != "batch_2" || campaign.FailedCount != 2 {
			t.Errorf("unexpected campaign: %+v", campaign)
		}
	})
}

const wireCampaignPreview = `{"totalRecipients":5,"estimatedCredits":10,"optedOutCount":1,"invalidCount":0,"invalidNumberCount":1,"landlineCount":1,"sampleRecipients":[{"phone":"+15551230001","name":"Ada"}],"blockedCount":1,"sendableCount":4,"byCountry":{"US":{"count":4,"credits":10,"allowed":true},"GB":{"count":1,"credits":0,"allowed":false,"blockedReason":"International messaging is not enabled"}},"warnings":["1 recipient cannot be reached with your current verification: GB (1)"],"messagingProfile":{"canSendDomestic":true,"canSendInternational":false,"verificationType":"toll_free","verificationStatus":"approved"},"recipientCount":5,"currentBalance":100,"hasEnoughCredits":true}`

func TestCampaignsPreview_DecodesThePreview(t *testing.T) {
	server := campaignServer(t, map[string]string{"GET /campaigns/camp_1/preview": wireCampaignPreview})
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	preview, err := client.Campaigns.Preview(context.Background(), "camp_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.RecipientCount != 5 || preview.EstimatedCredits != 10 {
		t.Errorf("expected RecipientCount 5 and EstimatedCredits 10, got %d and %v", preview.RecipientCount, preview.EstimatedCredits)
	}
	if preview.SendableCount == nil || *preview.SendableCount != 4 || preview.BlockedCount == nil || *preview.BlockedCount != 1 {
		t.Errorf("expected SendableCount 4 and BlockedCount 1, got %v and %v", preview.SendableCount, preview.BlockedCount)
	}
	if us := preview.ByCountry["US"]; !us.Allowed || us.Count != 4 || us.Credits != 10 {
		t.Errorf("unexpected ByCountry[US]: %+v", us)
	}
	if gb := preview.ByCountry["GB"]; gb.Allowed || gb.BlockedReason != "International messaging is not enabled" {
		t.Errorf("unexpected ByCountry[GB]: %+v", gb)
	}
	if preview.MessagingProfile == nil || !preview.MessagingProfile.CanSendDomestic || preview.MessagingProfile.CanSendInternational {
		t.Fatalf("unexpected MessagingProfile: %+v", preview.MessagingProfile)
	}
	if preview.MessagingProfile.VerificationType == nil || *preview.MessagingProfile.VerificationType != "toll_free" {
		t.Errorf("expected VerificationType toll_free, got %v", preview.MessagingProfile.VerificationType)
	}
	if len(preview.Warnings) != 1 {
		t.Errorf("expected one warning, got %v", preview.Warnings)
	}
	if preview.CurrentBalance != 100 || !preview.HasEnoughCredits {
		t.Errorf("expected CurrentBalance 100 with enough credits, got %d/%v", preview.CurrentBalance, preview.HasEnoughCredits)
	}
	if preview.OptedOutCount != 1 || preview.InvalidCount != 0 || preview.InvalidNumberCount != 1 || preview.LandlineCount != 1 {
		t.Errorf("unexpected exclusion counts: %+v", preview)
	}
	if len(preview.SampleRecipients) != 1 || preview.SampleRecipients[0].Phone != "+15551230001" || preview.SampleRecipients[0].Name != "Ada" {
		t.Errorf("unexpected SampleRecipients: %+v", preview.SampleRecipients)
	}
}

func TestCampaignPreview_ReadsTotalRecipients(t *testing.T) {
	var preview CampaignPreview
	if err := json.Unmarshal([]byte(`{"totalRecipients":7,"estimatedCredits":14,"blockedCount":0,"sendableCount":7,"warnings":[]}`), &preview); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.RecipientCount != 7 || preview.EstimatedCredits != 14 {
		t.Errorf("expected RecipientCount 7 from totalRecipients and EstimatedCredits 14, got %d and %v", preview.RecipientCount, preview.EstimatedCredits)
	}
}

func TestCampaignPreview_RoundTrip(t *testing.T) {
	var original CampaignPreview
	if err := json.Unmarshal([]byte(wireCampaignPreview), &original); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded CampaignPreview
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}
