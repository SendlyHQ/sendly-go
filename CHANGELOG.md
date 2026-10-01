# sendly-go

## 4.3.0

### Minor Changes

- **Options and fields the API already supported, now in the Go SDK.**
  - `Webhooks.GetDeliveriesWithOptions(ctx, id, &ListWebhookDeliveriesOptions{Limit, Offset, Status})` pages through deliveries and filters them by status.
  - `Enterprise.Webhooks.SetWithOptions(ctx, &SetEnterpriseWebhookRequest{URL, Events, Workspaces})` chooses which events and workspaces the enterprise webhook receives. `EnterpriseWebhook` gains `SigningSecret`, returned only once, by the account's first `Set` or `SetWithOptions`, and `Events` and `Workspaces`.
  - `Enterprise.Workspaces.InheritVerificationWithOptions(ctx, id, &InheritVerificationRequest{SourceWorkspaceID, PurchaseNewNumber})`: with `PurchaseNewNumber`, the workspace does not share the source workspace's approval. Its business details are copied, and buying it a toll-free number and filing that number's toll-free verification are best effort: `TollFreeNumber` is nil when no number could be bought, and `Status` stays `pending` unless the verification was filed, which needs opt-in images and a valid business registration number. `InheritVerificationResponse` gains `NewNumber`, true when `PurchaseNewNumber` was set.
  - `Credits.BillingMode` is `"prepaid"` or `"pooled"`.
  - `CreateAPIKeyResponse.ExpiresAt` is when a key created with `CreateAPIKeyRequest.ExpiresAt` expires. `CreateAPIKeyResponse.APIKey` carries the new key's `Permissions`, the scopes it was granted; its doc said they were never returned.
  - `BatchMessageResponse` gains `ID`, `Delivered`, `CreditsReserved` and `CreditsRefunded`, and on a send `OptedOutSkipped`, `InvalidSkipped` and `Retrying`.
  - `BatchPreviewResponse` gains the fields the preview returns: `Total`, `Sendable`, `Duplicates`, `CreditBalance`, `HasSufficientCredits`, `Pooled`, `KeyType`, `KeyScopes`, `HasWriteScope`, `MessagingProfile`, `ByCountry`, `BlockedMessages`, `Compliance` and `Warnings`.
  - `ListMessagesResponse.Pagination` carries `Total`, `Limit`, `Offset`, `Page`, `TotalPages` and `HasMore`. `Count` is the number of messages in the page, as it always was; its doc said otherwise.
  - `GroupMessageResponse.Recipients` lists each `GroupRecipient` with its status on a live group send.
  - `Campaign` gains `BatchID`, `FromSender` and `TargetType`. `CampaignPreview` gains `CurrentBalance`, `HasEnoughCredits`, `OptedOutCount`, `InvalidCount`, `InvalidNumberCount`, `LandlineCount` and `SampleRecipients`. `CampaignStatusCompleted` is the status of a campaign that has been sent.
  - `WebhookTestResult` gains `Message` and `Delivery`. `WebhookSecretRotation` gains `RotatedAt`, `GracePeriodHours` and `NewSecretVersion`.
  - `AnalyticsOverview` gains `DeliveryRatePercent`, `TotalWorkspaces`, `SuspendedWorkspaces`, `TotalDelivered`, `TotalFailed` and `TotalCredits`. `AnalyticsCreditsResponse` gains `TotalBalance`, `TotalLifetime`, `TotalUsed` and `WorkspaceCount`.
  - `TransactionTypeTransfer`, `TransactionTypeAdminGrant` and `TransactionTypeAdminSeed`, which the API records. Auto-recharges are recorded as `TransactionTypePurchase`.
  - `CallErrorCodeFromNumberNotSupported`: `Calls.Create` answers 400 `from_number_not_supported` when `From` is not a US or Canadian number.
- **WhatsApp extras.**
  - `WhatsApp.Senders.UploadProfilePhoto(ctx, phoneNumber, filename, file)` sets a sender's profile photo (JPEG or PNG, up to 5 MB, sent once and never retried, not even after a timeout or network error) and `DeleteProfilePhoto` removes it. Both return the updated `WhatsAppSenderProfile`.
  - `WhatsApp.Senders.GetConversationalComponents` and `UpdateConversationalComponents` read and replace a sender's ice breakers and commands. In `UpdateWhatsAppConversationalComponentsRequest` a nil list is left unchanged and an empty list clears the stored one.
  - `WhatsApp.Senders.SetCalling(ctx, phoneNumber, enabled)` switches WhatsApp calling on or off and returns a `WhatsAppSenderCalling`. Inbound WhatsApp calls then ring like phone calls; there is no API for placing them.
  - `WhatsAppSender` gains `BusinessAccountID`, `BusinessName`, `CallingEnabled` and `OutboundCallingAllowed`.
  - `WhatsApp.Signup.CreateWithOptions(ctx, &CreateWhatsAppSignupRequest{PhoneNumber, BusinessAccountID, VerificationMethod, DisplayName})` adds a number to a WhatsApp Business Account the workspace already connected, without the Facebook step: WhatsApp sends the number a code, `Signup.Verify(ctx, id, code)` submits it and `Signup.Resend(ctx, id, verificationMethod)` asks for another. With only `PhoneNumber` it sends what `Create` sends. With `BusinessAccountID` a 5xx, a timeout or a network error is returned without a retry, because the request may have run and each retry could start, and charge, a new signup. `CreateWithOptions` returns a `*ValidationError` for a whitespace-only `BusinessAccountID`, before anything is sent, instead of starting a paid Facebook signup. `Verify` also returns a 5xx, a timeout or a network error without a retry, because each retry submits the code again and can use up one of its 5 tries. Both still retry a retryable 429, which the API refused before running. Nor does Go's HTTP transport resend these two requests or a profile photo upload when a reused keep-alive connection drops.
  - Signups can be `"verifying"`. `WhatsAppSignup` gains `VerificationMethod`, `VerificationAttemptsRemaining` and `VerificationCode` (the code once its text has arrived on the number), and `WhatsAppSignupSession` gains the signup fields a number added by code returns. New failure reasons: `verification_start_failed`, `verification_failed` and `verification_expired`. `WhatsAppSignupStatus*` and `WhatsAppVerificationMethod*` constants name the values.
  - `WhatsAppErrorCode*` constants name the error codes of the new WhatsApp methods, `whatsapp_send_failed` and the new `whatsapp_send_unconfirmed`.
  - `Call` and `WebhookCallData` gain `Channel`, a `CallChannel`: `CallChannelPhone`, `CallChannelWhatsApp` or `CallChannelBrowser`. Any other value decodes as itself.
- `Enterprise.Workspaces.ProvisionBulk` accepts up to 100 workspaces, the API's limit. It refused more than 50 before sending anything.
- New types: `ListWebhookDeliveriesOptions`, `SetEnterpriseWebhookRequest`, `InheritVerificationRequest`, `GroupRecipient`, `ListMessagesPagination`, `BatchPreviewMessagingProfile`, `BatchPreviewCountry`, `BatchPreviewBlockedMessage`, `BatchPreviewCompliance`, `BatchPreviewShaftBlock`, `BatchPreviewQuietHoursBlock`, `CampaignSampleRecipient`, `WebhookTestDelivery`, `CreateWhatsAppSignupRequest`, `WhatsAppCommand`, `WhatsAppConversationalComponents`, `UpdateWhatsAppConversationalComponentsRequest`, `WhatsAppSenderCalling` and `CallChannel`.

### Patch Changes

- **A retried 5xx keeps its idempotency key.** After a 5xx the client sent the retry with a new auto-generated key, a leftover from when the API recorded server errors under the key. The API has not recorded a 5xx since August, so the retry runs again under the same key either way. A new key only lost protection in one case: when the API had finished the request and recorded its answer but a gateway returned the 5xx, a retry with a new key sent the message again. The retry now carries the same key, so that case returns the recorded answer instead.
- **A 429 is waited out only when waiting can help, and never for more than a minute.** The client waited out and retried every 429. It now retries only an ordinary `rate_limit_exceeded`, the per-minute `provision_rate_limit` from enterprise workspace provisioning (120 a minute), a 429 with no code, or `too_many_concurrent_verifications` (too many first-time API key checks at once), and only when the wait is 60 seconds or less. When there is no `Retry-After` header, the wait is read from the body's `retryAfter`, which fills `RetryAfter` too. A waited-out 429 is sent again after exactly that wait, with no backoff added, and after the last attempt the client returns without waiting. Every other 429 is returned at once as a `*RateLimitError` whose `Code` names it:
  - `too_many_failed_key_attempts`: repeated wrong API keys from one address locked the account out for up to 5 minutes, and the call waited that out before failing. Fix the key, then wait, since until the lockout ends the right key can be refused too.
  - `rate_limit_exceeded` from `Verify.Send` and `Verify.Resend` against the per-phone limit (5 codes per 10 minutes) or the daily limit (20 per day): the request was sent three more times. It now returns at once with `RetryAfter` set when the wait is over a minute; a wait of a minute or less, near the end of the window, is waited out like an ordinary rate limit.
  - `max_attempts_exceeded` from `Verify.Check`, `daily_call_limit` from `Calls.Create` and `quota_exceeded` were all retried for about 7 seconds before the same answer.
  - The hourly `provision_rate_limit` (1,000 workspaces an hour) was retried for about 7 seconds too. It now comes back with `RetryAfter` set, so you can pace provisioning.

  `Error()` for a 429 returned at once includes the API's message and code, plus the wait when there is one, so a logged lockout says what went wrong rather than only when to retry. An ordinary rate limit's `Error()` is unchanged when the response carries a `Retry-After` header; when its wait comes only from the body, it now prints that wait. `Media.Upload`, `Enterprise.UploadVerificationDocument` and `BusinessUpgrade.Start` and `Resubmit` still send their uploads once, so any 429 there, `too_many_concurrent_verifications` included, is returned at once.
- **Methods that failed on every call now work.** Each was checked against the handler it calls.
  - `Webhooks.GetDeliveries` failed with `failed to unmarshal response`, after three retries: the API wraps the deliveries in `{deliveries, pagination}`. A delivery that got no response, such as one that timed out, has a nil `ResponseTimeMs`.
  - `Campaigns.Schedule` sent `scheduled_at`, and the API reads `scheduledAt`, so every call got a 400 `scheduledAt is required`.
  - `Media.Upload` and `Enterprise.UploadVerificationDocument` labelled the file `application/octet-stream`, which the API's upload filters refuse with a 500. The file is now labelled with the type its content shows, or failing that the type its filename's extension names, and keeps its filename.
  - `Messages.SendGroup` failed after a live group send had gone out and been charged, because a live send lists its recipients as objects. `To` still holds the phone numbers, and `Recipients` has each recipient's status.
  - `Enterprise.Analytics.Overview` failed whenever the delivery rate had decimal places (66.67, say). `DeliveryRate` holds the rate rounded to a whole number, and `DeliveryRatePercent` the exact value.
- **Values that were wrong on every call.**
  - `Account.GetCredits` reported `ReservedBalance` and `AvailableBalance` as 0: the API sends them in camelCase.
  - `Account.RotateAPIKey` left `Prefix` empty and `Permissions` nil on `NewKey` and `OldKey`: the rotation returns the key records, which name them `keyPrefix` and `scopes`. Both keys now carry them as `GetAPIKey` reports them, with `Prefix` ending in `...`.
  - `Messages.GetBatch` and `Messages.ListBatches` left `BatchID` empty, because those endpoints call it `id`. `BatchID` and the new `ID` are filled from either name.
  - `Messages.PreviewBatch` filled only `Blocked` and `CreditsNeeded`. `TotalMessages`, `WillSend`, `CurrentBalance`, `HasEnoughCredits` and `BlockReasons` are now filled from what the preview returns. `CanSend` is true when the preview found nothing that stops a send: something is sendable, the batch has at most 10,000 messages, nothing is blocked except opted-out recipients (a live send skips those but rejects the whole batch for any other block), the key has the `sms:send` scope, and the balance covers the credits needed unless the key is a test key. Blocks are judged as a live send judges them: a test key's send skips the verification and destination checks, so it can go through while `CanSend` is false. A send can also be refused for what the preview does not check, such as a suspended workspace or the monthly message quota, and it charges a repeated recipient for every message where the preview counts it once.
  - Every campaign method read snake_case keys the API does not send, so the counts were 0 and `EstimatedCredits`, `CreditsUsed`, `ScheduledAt`, `StartedAt` and `CompletedAt` were nil. `Campaign` reads the campaign as the API returns it (`StartedAt` is when the send began), and still reads its snake_case tags, so marshalling a `Campaign` is unchanged. `Campaigns.Preview` filled only `Warnings`; `CampaignPreview`, `CountryAccessInfo` and `MessagingProfileAccess` read the preview the same way.
  - `Campaigns.Send` returned a campaign with no ID or name: the API answers with the batch the messages went out in. It now reads the campaign back after the send and returns it, `BatchID` included. `Status` is still the batch's status as the send reported it (`failed` when every message failed, `partial_failure`, `processing`), not the campaign's, which is `completed` after any accepted send; `Get` returns that.
  - `Enterprise.Analytics.Credits` returned an empty `Data` and nothing else; it returns the credit totals.
  - `Enterprise.Webhooks.Set` dropped the signing secret, which the API returns only once, on the first registration, so the secret could not be read without rotating it.
  - `Webhooks.Test` left `StatusCode`, `ResponseTimeMs` and `Error` nil; they are read from the test delivery. `Webhooks.RotateSecret` returned a zero `Webhook` (with a `Mode` of `"all"` that the API never sent); it now carries the webhook's `ID`, which is all the rotation response includes.
  - `Conversations.AddLabels` and `Conversations.RemoveLabel` returned a zero `Conversation`: the API answers with the labels and with 204. Both read the conversation back after the change and return it.
- **A response that does not decode is returned at once.** A 2xx body that did not fit the result type came back as a `*NetworkError` and was retried three more times over about seven seconds, and a POST without server-side idempotency ran again each time. It is returned after the first attempt: the same `*NetworkError`, with the same message and the JSON error in `Err`.
- **Docs that described behaviour the API does not have.** `Webhooks.Backfill` no longer says synthesized events get fresh ids and to dedupe on `data.object.id`: they carry the event id the original dispatch used, so dedupe on `event.id`. `WebhookSecretRotation` described an old secret that stays valid for a while; deliveries are signed with the new secret as soon as the rotation returns, so have your endpoint accept both secrets while you deploy the new one, as `Webhooks.RotateSecret` now says. `Webhooks.Test` says a failed test, including one for a webhook that does not exist, comes back as a `*ValidationError` whose `Message` says why. `Media.Upload` names the types it accepts and the errors other files get. `CreditTransaction.MessageID` says the API never fills it. `CreateAPIKeyRequest` says a key created without `Scopes` gets the scopes of the key that creates it, not a standard set, and that asking for a scope that key lacks is refused. `ListMessagesRequest.Limit` and `ListBatchesRequest.Limit` default to 50, not 20, and `ListBatchesResponse.Count` is the number of batches in the page, not the total.
- **WhatsApp send failures.** `Messages.SendWhatsApp` documents a 502 `whatsapp_send_failed` as a message that provably never reached the carrier, so it was not sent and is safe to send again, and the new 409 `whatsapp_send_unconfirmed`: the outcome is unknown, and the message was marked failed and refunded but may still be delivered, so check before sending it again (it could arrive twice). The client does not retry a 409, as with every 4xx.
- **WhatsApp docs match the API.** The doc comments now say that the API never sends the `"expired"` signup status, that a closed window returns its past `ExpiresAt` rather than nil, that a media send returns its caption as `Text`, how in-window replies are priced (1 credit for the first 1,000 per sending number each month, then the destination's utility price), which roles and scopes connecting and editing need, the `waba_mismatch` and `registration_timeout` failure reasons, `template_header_variable_unsupported`, `whatsapp_unavailable` (503), `whatsapp_signup_limit_reached` (429), and `whatsapp_send_failed` as a final 422 or a retried 502. Nothing changes at runtime.

### Security

- **Webhook and API key ids are percent-encoded.** 4.0.0 said every id is encoded before it goes into the path, but `Webhooks.Get`, `Update`, `Delete`, `Test`, `ResetCircuit`, `Redeliver`, `Backfill` and `RotateSecret`, and `Account.GetAPIKey`, `GetAPIKeyUsage`, `RevokeAPIKey` and `RotateAPIKey`, still pasted it in as given, so an id such as `whk_1/../../account/keys` or `key_1?x=1` reached a different endpoint with your API key. They use `url.PathEscape` like every other method. Ordinary ids are sent byte-for-byte as before.
- **An id of `""`, `.` or `..` is refused before anything is sent.** Percent-encoding leaves dots alone, and the URL is resolved as a dot-segment before it reaches the API, so `Enterprise.Workspaces.RevokeKey(ctx, "ws_1", "..")` sent `DELETE /enterprise/workspaces/ws_1/`, which deletes the workspace. An empty id likewise reached the endpoint one level up, so `Contacts.Get(ctx, "")` asked for the contact list. Every method now returns a `*ValidationError` for such an id without sending a request. Ids that merely contain dots, such as `a.b`, are sent as before.

### Deprecated

Every member below is kept, so existing code compiles. Each is marked `Deprecated:`.

- `AnalyticsOverview.DeliveryRate`: rounded to a whole number. Use `DeliveryRatePercent`. It becomes a `float64` in the next major version.
- `AnalyticsCreditsResponse.Data` and `AnalyticsCreditDay`: the endpoint returns totals, not a daily series, so `Data` is always empty.
- `BatchPreviewResponse.TotalMessages`, `WillSend`, `CurrentBalance` and `HasEnoughCredits`: filled now, and the same values as `Total`, `Sendable`, `CreditBalance` and `HasSufficientCredits`. `BatchPreviewResponse.Messages` and `BatchPreviewItem`: the preview has no per-message list; use `BlockedMessages`.
- `CampaignStatusSent` and `CampaignStatusPaused`: no campaign has either status. A sent campaign is `CampaignStatusCompleted`, and `Campaigns.List` treats a `sent` filter as `completed`.
- `Campaign.TemplateID`: always nil. `CampaignPreview.EstimatedCost`: always 0; use `EstimatedCredits`.
- `WebhookSecretRotation.OldSecretExpiresAt`: always empty. The old secret stops matching as soon as the rotation returns.
- `TransactionTypeAdjustment`: no transaction has this type.

### Upgrade notes

No signature changed, so existing code compiles unchanged.

- `BatchMessageResponse`, `GroupMessageResponse`, `BatchPreviewResponse`, `WebhookTestResult`, `AnalyticsOverview`, `Campaign`, `CampaignPreview`, `CountryAccessInfo` and `MessagingProfileAccess` gained an `UnmarshalJSON` method. A struct of yours that embeds one of them as an anonymous field inherits that method, so `encoding/json` decodes the whole struct through it and your own fields stay empty, with no compile error. Give the embedded type a named field instead.
- `Campaigns.Send`, `Conversations.AddLabels` and `Conversations.RemoveLabel` make a second request to read the result back (reading a campaign needs the `campaigns:read` scope, a conversation `sms:read`). If that read fails, the change has still been made, so they return a `Campaign` or `Conversation` holding what they know (`ID`, and for a send the batch, its status and its counts) and no error.
- `BatchPreviewResponse.CanSend` was always false. It is now true when the preview found nothing that stops a send, so code that branches on it takes the other branch for a sendable batch.
- `WebhookSecretRotation.Webhook.Mode` is empty; it used to be a made-up `"all"`.

## 4.1.0

### Patch Changes

- **4xx responses are no longer retried.** The client used to retry every status it had no typed error for, so a 409 `lines_busy`, 428 `e911_required`, 403 `live_key_required` or 409 `rcs_field_locked` went through the full backoff (three more attempts, about seven seconds) before the `*SendlyError` reached you. Any 4xx now returns at once; only 429 (`*RateLimitError`), 5xx responses, timeouts and network errors retry as before. If you were passing `WithMaxRetries(0)` to get the refusal quickly, you can drop it.

## 4.0.0

### Breaking Changes

- **The module path is now `github.com/SendlyHQ/sendly-go/v4`.** Go carries the major version in the import path, so upgrading means changing your imports:

  ```sh
  go get github.com/SendlyHQ/sendly-go/v4
  ```

  ```go
  import "github.com/SendlyHQ/sendly-go/v4/sendly"
  ```

  No exported identifier was removed or renamed, so after the import rewrite your code compiles unchanged. Every v3 release stays available under the old path.

- **Only message events get a message view.** When a webhook is parsed, `event.Data` is filled only for `message.*` events, and not for `message.opt_in` or `message.opt_out`, which carry an opt-out record rather than a message. For every other event `event.Data` is now zero-valued: read the payload from `event.RawObject`, or decode it with `event.DecodeObject(&v)`. This fixes two defects. A lifecycle event whose object reused a message field name at a different type (a nested `status`, a numeric `id`) made the whole event fail to parse, `RawObject` included. And the legacy `message_id` fallback filled a non-message event's id, so `contact.auto_flagged` reported the contact's id as a message id. A `message.*` event whose object does not decode still returns an error.

### Security

- **Path parameters are percent-encoded.** Every id you pass is now encoded (`url.PathEscape`) before it goes into the request path. An id containing `/`, `?` or `#` used to change which endpoint the request reached: an id of `../../account/keys` left its collection and hit another endpoint carrying your API key. Ordinary ids are sent byte-for-byte as before.

## 3.40.0

### Minor Changes

- **Lifecycle webhook payloads are reachable.** `ParseEvent` decoded every
  `data.object` into `WebhookMessageData`, which is correct for `message.*` and
  wrong for `rcs_*`, `whatsapp_*`, `call.*`, `brand.*`, `campaign.*`,
  `assignment.*`, `number.*`, `port*` and `contact.*` — those carry a different
  object entirely, so the struct came back with every field at its default and
  **no error was raised**. An integration looked healthy while dropping `agent_id`
  and `stage`. `WebhookEvent` gains `RawObject` (the payload exactly as it
  arrived, for every event type) and `DecodeObject(&v)` to unmarshal it into a
  type of your choosing.

- **A lifecycle event can no longer fail the parse.** The message decode is now
  attempted only for events that carry a message, and never fails the event: a
  payload reusing a message field name at a different type (a nested `status`
  object, a numeric `id`) previously made the whole event unreadable, `RawObject`
  included.

- **`message.opt_in` and `message.opt_out` are no longer decoded as messages.**
  They share the prefix but carry an opt-out record
  (`{phone_number, keyword, from_number, timestamp}`), so the message view for
  them was entirely empty.

- **Every event type the API emits is now declared**, including `conversation.*`,
  `draft.*`, `rcs_*`, `whatsapp_*` and `call.*`, which were all missing.

### Upgrade notes

Nothing was removed and no signature changed, so existing code compiles
unchanged. Two behavior changes are worth checking a handler against.

- **`event.Data` is now zeroed for every non-message event.** Previously it was
  populated but wrong, and where a lifecycle object happened to reuse a message
  field name the wrong value came through under a message's meaning:
  `contact.auto_flagged` carries the **contact** id in `id`, which `Data.ID`
  reported as the message id, so a handler keyed on it acted on the wrong
  record. If you read `event.Data` on anything but `message.*`, move to
  `DecodeObject`. On `contact.auto_flagged` the message that triggered the flag
  is the payload's own `message_id` field, not `id`.

- **Handlers will start seeing events that never arrived before.** A lifecycle
  payload whose field collided with a message field's type — a nested `status`
  on `call.completed`, a numeric `id` on `number.activated` — used to fail
  `ParseEvent` outright, so your handler never got an event to dispatch on.
  They parse now, which means a `default` branch that has never run may start
  running.

### Deprecated

- `WebhookEventMessageQueued` and `WebhookEventMessageUndelivered`. The API has
  never emitted these and rejects them with a 400 when you subscribe. They are
  kept for one more cycle and will be removed in the next major.

## 3.39.0

### Minor Changes

- **Error decoding is a package function, not a method on `APIError`.** `APIError` is embedded in `SendlyError`, `RateLimitError`, `AuthenticationError`, `ValidationError`, `NotFoundError` and `InsufficientCreditsError`. A custom `UnmarshalJSON` on it would be promoted into all six, so unmarshalling one of those would decode only the embedded fields and silently zero the wrapper's own (a `RateLimitError`'s `RetryAfter`, for instance). Decoding happens in the client instead, and `APIError` gains only the additive `Errors []APIFieldError` field.


## 3.38.0


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
