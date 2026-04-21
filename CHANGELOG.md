# sendly-go

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
