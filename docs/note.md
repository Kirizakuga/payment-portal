# Project Notes — payment-portal

> Last updated: 2026-05-15

---

## 🏗️ Architecture

- **Pattern**: Dependency Injection — `PaymentStore` is instantiated once in `main.go` and passed into `OrderController` and `WebhookController`. Never use global variables for state.
- **Templates**: Parsed once at startup via `template.Must(template.ParseGlob(...))`. Both controllers share the same `*template.Template` instance.
- **Single-instance only**: The in-memory `PaymentStore` only works correctly when running **one** server instance. If you scale horizontally (e.g., AWS App Runner, Render auto-scale), migrate state to **Redis or PostgreSQL** first.

---

## ⚠️ Known TODOs (Must Do Before Production)

### 1. Replace Log Stub with Real Database Write
**File**: `controllers/payment-controller.go` → `VerifyPaymentWebhookData`
```go
// TODO: Replace this log with a real write to Google Sheets / PostgreSQL.
log.Printf("PAYMENT SUCCESS: OrderCode=%d, Amount=%d VND. [TODO: Write to Google Sheets]", ...)
```
This is the **source of truth** write point. If this stays as a log, payments from users who close the tab will be permanently lost after 15 minutes (TTL eviction).

### 2. Protect `/admin/confirm-webhook`
**File**: `main.go`
```go
http.HandleFunc("/admin/confirm-webhook", controllers.ConfirmWebhook)
```
This route has **no authentication**. Anyone who discovers it can re-register your webhook URL to their own server and intercept payment events. Before going public:
- Add a secret token check: `?token=<SECRET>`, or
- Restrict to localhost / internal IP only, or
- Remove the route after first use.

### 3. Race Condition Fallback Has No Persistent Storage
**File**: `controllers/payment-controller.go`
```go
// WARNING: If order not found in store (race condition), payment is only logged.
// It is NOT written to a database. Add DB write here alongside the log.
log.Printf("PAYMENT RECORDED [RACE FALLBACK]: OrderCode=%d, Amount=%d", ...)
```

---

## 🔒 Security Decisions (Do Not Revert)

| Decision | Reason |
|---|---|
| Webhook always returns `200 OK` | Returning 4xx/5xx causes PayOS to retry for 3 days (self-DDoS) |
| Amount verified against `ExpectedAmount` in store | Prevents "Cheapskate Exploit" — user edits transfer amount in banking app |
| Idempotency check before writing | PayOS guarantees at-least-once delivery; prevents double DB entries |
| `payos.VerifyPaymentWebhookData()` called first | Cryptographic signature check — blocks forged webhook requests |

---

## 📐 OrderCode Rules

- Generated with `time.Now().UnixMilli() % 1_000_000` → **6-digit `int64`**
- The `% 1_000_000` modulo is **CRITICAL**: The `payos-lib-golang` library unmarshals response data into `map[string]interface{}`, making numbers `float64`. It then uses `fmt.Sprintf("%v")` to build the string for signature verification.
- In Go, `float64` values $\ge 1,000,000$ are formatted using scientific notation (e.g., `1.234567e+06`), which breaks the signature string compared to what PayOS actually sent.
- **Limit**: To avoid this bug, `OrderCode` must be $\le 6$ digits and `Amount` should be $< 10,000,000$ VND.
- Do **not** use 7+ digits until the library is fixed.


---

## 🔁 Polling Behaviour (Frontend)

- Polls `/api/check-status?orderId=...` every **3 seconds**
- Hard stops after **200 polls = 10 minutes** (standard VietQR transaction lifespan)
- Handles 3 states: `PENDING` (keep polling), `PAID` (redirect), `PARTIAL_PAYMENT_FRAUD` (show error)
- Webhook is source of truth — polling is UX only. A closed tab does not lose the payment.

---

## 📦 Webhook Registration (One-Time Setup)

After deployment, call this once to register your URL with PayOS:
```
GET https://your-domain.com/admin/confirm-webhook?url=https://your-domain.com/payos-webhook
```
PayOS stores this permanently. Only redo if your domain changes.

---

## 🚀 Scaling Checklist (When Ready)

- [ ] Replace `store/store.go` in-memory map with Redis (`go-redis/redis`)
- [ ] Replace `log.Printf` stub in webhook handler with Google Sheets / PostgreSQL write
- [ ] Add authentication to `/admin/confirm-webhook` or remove it post-setup
- [ ] Add structured logging (e.g., `log/slog`) to replace raw `log.Printf`
- [ ] Add unit tests for `store/store.go` TTL eviction and `WebhookController` security pipeline
$note
