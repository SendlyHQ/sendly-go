# sendly-go

## 3.38.0

### Minor Changes

- **`client.Templates` and `client.Verify` now actually reach the API.** All 17 methods across `TemplatesService`, `VerifyService` and `SessionsService` handed a bare path such as `/templates` to an internal helper that expects a fully-qualified URL. The request never left the process: every call failed with a `*NetworkError` wrapping `unsupported protocol scheme ""`, on the first attempt, with no HTTP traffic and no retry. They now build URLs the way the working services do, and they go through the client's rate limiter and retry policy like every other call. This is the important line in this release: if you wrote code against `Templates.List`, `Presets`, `Get`, `Create`, `Update`, `Publish`, `Preview`, `Delete`, `Generate`, or against `Verify.Send`, `Resend`, `Check`, `Get`, `List`, `Verify.Sessions.Create`, `Verify.Sessions.Validate`, those calls were inert and are now live. `Create`, `Update`, `Publish` and `Delete` will really write, and `Verify.Send` and `Resend` will really send a code and consume credits. Template calls need the `templates:read` / `templates:write` scope on the key (`Templates.Generate` needs `sms:send`), verify calls need `verify:send` / `verify:read`; all of these are in the default scope set for a new key.
- **`Templates.Clone` is the one exception and still does not work.** It was repointed with the rest, but the versioned API serves no clone route, so it returns a `*NotFoundError`. To copy a template today, read it with `Get` and pass its `Text` to `Create`. The method and `CloneTemplateRequest` are kept so existing code compiles.
- **API key management repointed onto routes the server serves.** `ListAPIKeys`, `GetAPIKey` and `GetAPIKeyUsage` requested `/keys...`, a path with no route at all, so they always came back as a 404 `*NotFoundError`. They now call `/account/keys...`. `RevokeAPIKey` sent `DELETE /account/keys/{id}`, a verb that path does not accept; revocation is now `PATCH /account/keys/{id}/revoke`. All four work for the first time. Note that the server refuses to revoke the key the client is currently authenticating with, and returns a 400 if you try.
- **`CreateAPIKey` could never succeed either.** The body carried only `name`, and the endpoint requires a `type`, so every call returned a 400 `*ValidationError` reading "Name and type are required". `CreateAPIKeyRequest` gains `Type` ("test" or "live", defaulting to "test" when empty, rejected client-side when it is anything else) and `Scopes` (leave nil for the standard set). `CreateAPIKey(ctx, name)` still creates a test key. Creating a live key additionally requires a verified business and a positive credit balance, otherwise the server answers 403 or 402.
- **`ListAPIKeys` and `GetCreditTransactions` decode the real response.** Both expected a bare JSON array. The API returns `{"keys": [...]}` and `{"transactions": [...]}`, so both failed to unmarshal and returned an error on every call. They unwrap the envelope now. Key fields are also read from the names the API really sends: `APIKey.Permissions` comes from `scopes`, `IsRevoked` is derived from `isActive`, and the timestamps come from the camelCase keys, so those fields are populated rather than zero.
- **`Account.Get` reads the account out of its envelope.** The account arrives nested under `user`, so `Account.ID`, `Email` and `CreatedAt` were always empty even when the request succeeded. They are filled in now. If a response arrives with no user block, `Get` returns an `invalid_response` error rather than a blank `Account`.
- **`GetAPIKeyUsage` returns the fields the endpoint actually sends:** `KeyName`, `Summary` (`TotalRequests`, `TotalCredits`, `LastUsed`), `RecentRequests` (up to 20 recent calls with endpoint, method, status code, credits and timestamp) and `EndpointBreakdown` (call counts per endpoint). Usage is reported per API request, not per message. `GetAPIKey` and `GetAPIKeyUsage` also reject an empty key ID locally instead of requesting a malformed path.
- **Automatic idempotency keys on every POST.** The client now generates an `Idempotency-Key` per logical request and reuses it across its own retry attempts, so on endpoints that support idempotency the server recognizes a retry of a request that already got through and replays the original result instead of executing it twice. That narrows the duplicate-send and double-charge window that timeout retries used to open, it does not close it: the server records a key only once the first attempt has finished, so a retry that fires while the original is still running is not seen as a repeat. Keys are rotated after a 5xx (the outcome is known, so the retry should re-execute) and preserved across timeouts and network errors (the outcome is unknown, so the server should dedupe). Nothing about your call sites changes.
- **Optional caller-supplied keys** through six new variants: `Messages.SendWithOptions`, `SendWhatsAppWithOptions`, `SendRcsWithOptions`, `SendGroupWithOptions`, `ScheduleWithOptions` and `SendBatchWithOptions`, each taking trailing `RequestOption` values. Pass `WithIdempotencyKey("order-4821-shipped")` when you need idempotency across process restarts or your own retry loop: repeating the request with the same key inside 24 hours returns the original response instead of executing again. Keys are validated locally (1 to 255 printable ASCII characters); empty and whitespace-only values are treated as absent and fall back to the automatic key. Two things worth knowing: the response is cached under the key once the first attempt completes, including error responses, so retrying a failed call with the same key returns the recorded failure and you need a fresh key to re-execute; and reusing a key with a different request body is rejected with a 422.
- **`Messages.SendBatch` deliberately sends no automatic key.** The batch endpoint already dedupes header-less retries server-side by hashing the request content, and an automatic key would bypass that. A key you supply yourself through `SendBatchWithOptions` is always sent.
- **Multipart uploads carry a single-use key per attempt:** `Media.Upload`, `Enterprise.UploadVerificationDocument`, `BusinessUpgrade.Start` and `BusinessUpgrade.Resubmit`.
- **No signatures changed.** `Messages.Send`, `SendWhatsApp`, `SendRcs`, `SendGroup`, `Schedule` and `SendBatch` keep the exact parameter lists they had before this release, so method values, function-typed fields and interfaces that name them still compile. The per-request options live only on the `*WithOptions` variants.
- New exported API: `RequestOption`, `WithIdempotencyKey`, `APIKeyUsageSummary`, `APIKeyUsageRequest`, `APIKeyUsageEndpoint`, and the six `*WithOptions` message methods.

### Deprecations

Every member below was restored rather than removed, so existing code keeps compiling. Each is marked `Deprecated:` and will be reported by staticcheck and by editors using gopls.

- `Account.Name`: always nil. The account payload carries no display name. Use `Email` to identify the account.
- `APIKey.LastFour`: always empty. The API returns only the leading prefix of a key, never the tail. Use `Prefix`.
- `CreateAPIKeyResponse.APIKey`: mirrors the new flat `ID`, `Name`, `Type`, `KeyPrefix` and `CreatedAt` fields. Its own `Permissions` and `LastFour` stay empty, because a create response carries neither. Read the flat fields instead.
- `APIKeyUsage.MessagesSent`, `.MessagesDelivered`, `.MessagesFailed`: always 0. Usage is counted per API request, not per message. Use `Summary.TotalRequests` for call volume, or `Messages.List` and count by status.
- `APIKeyUsage.CreditsUsed`: mirrors `Summary.TotalCredits`. Use `Summary.TotalCredits`.
- `APIKeyUsage.PeriodStart`, `.PeriodEnd`: always empty. The endpoint covers the most recent requests rather than a billing period. Use `RecentRequests[].CreatedAt` for the window it covers and `Summary.LastUsed` for the latest activity.
- `TemplatePreview.ID` and `.PreviewText`: mirror the new `TemplateID` and `RenderedText`. `TemplatePreview` also gains `CharacterCount` and `SegmentCount`.
- `TemplatePreview.Name` and `.Variables`: always empty. A preview response carries neither. Read them from `Templates.Get`.

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
- `useCase` validation expanded from 23 entries to the full 43-value carrier use-case enum.

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
