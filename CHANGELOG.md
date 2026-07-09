# sendly-go

## 3.36.0

### Minor Changes

- New `client.TenDlc` resource for 10DLC local-number texting registration — register your business for carrier review and text from local (10-digit) US numbers, now programmatic. The full flow: register a brand (`CreateBrand`), poll it to `verified` (`GetBrand`), pre-check a use case (`Qualify`), create a campaign (`CreateCampaign`), poll it to `active` (`GetCampaign`), then attach a number you own (`AssignNumber`); `ListBrands` / `ListCampaigns` / `ListAssignments` round out the surface. Statuses, throughput tiers, and failure reasons come back in plain language. Writes require a live API key with the `tendlc:write` scope.
- New exported types for brands, campaigns, qualification, and assignments (`CreateTenDlcBrandRequest`, `TenDlcBrandResponse`, `CreateTenDlcCampaignRequest`, `TenDlcCampaignResponse`, `TenDlcQualifyResponse`, `TenDlcAssignmentResponse`, and the corresponding list responses).

## 3.35.0

### Minor Changes

- Numbers: the programmatic buy flow now supports document-required countries end to end. `Numbers.Buy` returns a `documents_required` (or `payment_required`) status with a hosted-page action — relay the URL + code to the user to provide their business details and upload documents — and a re-buy with `ActionCode` set returns the new `under_review` status: the number is reserved and being verified + registered, and cannot send until it is active.
- Numbers: the owned-number listing (`Numbers.List`) now surfaces lifecycle fields — `RequirementsSubmittedAt`, `PendingCancellation`, and `ScheduledReleaseAt` — alongside the existing `Status` and `MonthlyCostCents`, so you can tell an active number from one that still needs documents, is under carrier review, or is scheduled for release.
- Messages: send from a number you own. Pass an owned, active number in E.164 as `From` on `Messages.Send` and the message goes out from that number. `From` can also be an alphanumeric sender ID for international destinations. It's optional and backward-compatible: omit it to use your default sender.

## 3.34.0

### Minor Changes

- New `client.Numbers` resource — buy and manage phone numbers programmatically. `ListCountries`, `ListAvailable`, `List` (owned), and `Buy`. When a country needs registration documents or a payment method, `Buy` returns a secure hosted-action hand-off: open the returned link, prove terminal access with the short code, complete the step on the Sendly dashboard, then re-call `Buy` with the `ActionCode` to provision.
- `Conversations.SuggestReplies(ctx, id)`: added the missing AI suggested-replies method for parity with the sibling SDKs.

## 3.33.0

### Patch Changes

- Version bump for the unified entity-upgrade coverage release (SDK + CLI + MCP + backend `cliAuthMiddleware`). No additional Go SDK code this cycle — the `client.BusinessUpgrade` resource shipped in Go 3.32.0.

## 3.32.0

### Minor Changes

- New `client.BusinessUpgrade` resource for the toll-free entity-upgrade ("fork-with-new-number") flow — when a customer forms a new legal entity (e.g. an LLC), reserve a new toll-free number under the new entity, submit it for carrier review, and atomically swap to it on approval without disrupting outbound SMS during the 1-2 week review window.
- Seven methods mirror the customer-facing API: `Preflight(ctx, *PreflightCandidate)`, `BestPrefill(ctx)`, `Start(ctx, workspaceID, *StartUpgradeParams, *EinDocument)`, `Status(ctx, workspaceID)`, `Cancel(ctx, workspaceID)`, `Resubmit(ctx, workspaceID, *StartUpgradeParams, *EinDocument)`, `SetDisposition(ctx, workspaceID, *SetDispositionRequest)`.
- Multipart EIN/CP-575 PDF upload via the new `EinDocument` struct (`Data []byte`, optional `Filename`, optional `ContentType`). Empty fields are dropped from the request body so `Resubmit` with a partial `StartUpgradeParams` only sends the fields you changed.
- New exported types: `PreflightCandidate`, `PreflightIssue`, `PreflightProposedFix`, `PreflightReport`, `StartUpgradeParams`, `EinDocument`, `StartUpgradeResponse`, `UpgradePending`, `UpgradeStatusResponse`, `CancelUpgradeResponse`, `ResubmitUpgradeResponse`, `DispositionResponse`, `BestPrefillFields`, `BestPrefillResponse`, `SetDispositionRequest`.

## 3.31.0

### Patch Changes

- Version bump for unified release. No Go SDK code changes — this release exists for parity with sibling SDKs that shipped fixes in this cycle (PHP doc/code mismatch, Ruby positional constructor, Rust + Java added `suggest_replies` / `suggestReplies`).

## 3.30.0

### Minor Changes

- `enterprise.Workspaces.SubmitVerification(ctx, workspaceID, *VerificationSubmitInput)`: rewritten to match the actual API shape (camelCase top-level, nested `address`/`contact` objects, `entityType` + `brn`/`brnType`/`brnCountry` instead of `businessType`/`ein`). The previous shape didn't match the server endpoint and always returned 400.
- **Partial-update friendly:** for resubmits on existing workspaces, send only the fields you want to change — everything else is filled from the existing record. Hosted page URLs (`/biz/`, `/opt-in/`, `/legal/`) generated during provision are auto-preserved.
- `enterprise.Workspaces.ResubmitVerification(ctx, workspaceID, *VerificationSubmitInput)`: convenience alias for resubmits — same as `SubmitVerification` but reads more naturally for one-field-change use cases.
- New exported types `VerificationSubmitInput`, `VerificationAddressInput`, `VerificationContactInput` — pointer-based fields (with `omitempty`) so unset values are dropped from the JSON body and the server merges with the existing verification record.

### Server-side fixes paired with this release

- `/api/v1/enterprise/workspaces/:id/verification/submit` now returns specific missing-field errors (e.g. `"Missing required fields: website"`) instead of listing every required field whether present or not.
- Endpoint accepts both flat and `{ verification: {...} }` wrapped shapes (matches `/enterprise/provision`).
- `useCase` validation expanded from 23 entries to the full 43-value Telnyx enum.

## 3.29.0

### Minor Changes

- `contacts.BulkMarkValid(ctx, BulkMarkValidRequest{IDs | ListID})`: clear the invalid flag on many contacts at once (up to 10,000 per call). Escape hatch for when auto-mark misclassifies at scale.
- Four new list-health `WebhookEventType` constants: `WebhookEventContactAutoFlagged`, `WebhookEventContactMarkedValid`, `WebhookEventContactsLookupCompleted`, `WebhookEventContactsBulkMarkedValid`.
- New `ListHealthEventSource` type with frozen constants (`ListHealthSourceSendFailure`, `ListHealthSourceCarrierLookup`, `ListHealthSourceUserAction`, `ListHealthSourceBulkMarkValid`) for the `source` field on auto-flag and mark-valid webhooks.
- `Contact` struct gains `UserMarkedValidAt` — when a user manually cleared an auto-flag. Carrier re-checks respect this timestamp and leave the contact clean.
- `CheckNumbersResponse` gains `AlreadyRunning` so the client knows when a rapid re-trigger was collapsed against an in-flight lookup.

## 3.28.0

### Minor Changes

- List-health: `contacts.MarkValid(ctx, id)` clears an auto-exclusion flag on a contact.
- List-health: `contacts.CheckNumbers(ctx, req)` triggers a background carrier lookup that flags landlines and non-SMS-capable numbers before you send. `CheckNumbersRequest{ListID, Force}` scopes the call.
- `Contact` struct gains `OptedOut`, `LineType`, `CarrierName`, `LineTypeCheckedAt`, `InvalidReason`, `InvalidatedAt` — all optional, populated by lookups or auto-flagged on terminal failures.

## 3.18.1

### Patch Changes

- fix: webhook signature verification and payload parsing now match server implementation
  - `VerifySignature()` accepts `timestamp string` parameter (empty string to skip) for HMAC on `timestamp.payload` format
  - `ParseEvent()` handles `data.object` nesting (with flat `data` fallback for backwards compat)
  - `WebhookEvent` adds `Livemode bool`, `Created interface{}` fields
  - `WebhookMessageData` renamed `MessageID` to `ID` (with `MessageID()` method alias)
  - Added `Direction`, `OrganizationID`, `Text`, `MessageFormat`, `MediaUrls` fields
  - `GenerateSignature()` accepts `timestamp` parameter
  - 5-minute timestamp tolerance check prevents replay attacks

## 3.18.0

### Minor Changes

- Add MMS support for US/CA domestic messaging

## 3.17.0

### Minor Changes

- Add structured error classification and automatic message retry
- New `ErrorCode` field with 13 structured codes (E001-E013, E099)
- New `RetryCount` field tracks retry attempts
- New `Retrying` status and `message.retrying` webhook event

## 3.16.0

### Minor Changes

- Add `TransferCredits()` for moving credits between workspaces

## 3.15.2

### Patch Changes

- Add Metadata field to BatchMessageItem

## 3.13.0

### Minor Changes

- Campaigns, Contacts & Contact Lists resources with full CRUD
- Template clone method
