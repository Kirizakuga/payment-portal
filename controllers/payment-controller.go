package controllers

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"payos-demo/store"

	"github.com/payOSHQ/payos-lib-golang"
)

// WebhookController holds a reference to the PaymentStore (Dependency Injection).
type WebhookController struct {
	Store     *store.PaymentStore
	Templates *template.Template
}

// VerifyPaymentWebhookData is the absolute source of truth for payment completion.
// It is the ONLY place that should trigger permanent business logic (e.g., writing to a database).
//
// CRITICAL: This handler MUST always return HTTP 200 OK to PayOS, even on errors.
// Returning 4xx/5xx causes PayOS to retry the webhook for up to 3 days (self-DDoS).
func (wc *WebhookController) VerifyPaymentWebhookData(w http.ResponseWriter, r *http.Request) {
	var webhookDataReq payos.WebhookType

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: could not read webhook body: %v", err)
		w.WriteHeader(http.StatusOK) // Always 200 to PayOS
		return
	}

	if err := json.Unmarshal(body, &webhookDataReq); err != nil {
		log.Printf("ERROR: could not unmarshal webhook body: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Step 1: Verify cryptographic signature using PAYOS_CHECKSUM_KEY.
	// This defends against forged webhook requests from external attackers.
	webhookData, err := payos.VerifyPaymentWebhookData(webhookDataReq)
	if err != nil {
		log.Printf("SECURITY: Webhook signature verification failed: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	orderCode := int64(webhookData.OrderCode)

	// Step 2: Idempotency check — prevents double-entry from PayOS retries.
	// Webhook systems guarantee "at-least-once" delivery, not exactly-once.
	session, found := wc.Store.Get(orderCode)
	if found && session.Status == store.StatusPaid {
		log.Printf("INFO: Duplicate webhook for already-paid order %d. Ignoring.", orderCode)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Step 3: Race condition safety — the webhook can arrive before CreatePaymentLink
	// finishes saving the session (especially on fast banking apps).
	// We never discard a verified payment just because our UI state hasn't caught up.
	if !found {
		log.Printf("WARNING: Webhook received for unknown order %d. Possible race condition. Recording payment anyway.", orderCode)
		// TODO: Push to permanent storage (Google Sheets / DB) with order details.
		// This is the fallback path for abandoned-tab payments.
		log.Printf("PAYMENT RECORDED [RACE FALLBACK]: OrderCode=%d, Amount=%d", orderCode, webhookData.Amount)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Step 4: Amount verification — defends against the "Cheapskate Exploit"
	// where a user manually edits the transfer amount in their banking app.
	// We still return 200 OK; returning an error would cause infinite PayOS retries.
	if webhookData.Amount != session.ExpectedAmount {
		log.Printf("FRAUD: Order %d expected %d VND but received %d VND. Flagging as PARTIAL_PAYMENT_FRAUD.",
			orderCode, session.ExpectedAmount, webhookData.Amount)
		wc.Store.UpdateStatus(orderCode, store.StatusFraud)
		w.WriteHeader(http.StatusOK) // Must be 200, not 400 — see comment above.
		return
	}

	// Step 5: All checks passed. This is the source-of-truth write point.
	// TODO: Replace this log with a real write to Google Sheets / PostgreSQL.
	// This must be done HERE, not in the frontend success page, so that
	// payments from closed-tab users are never lost.
	log.Printf("PAYMENT SUCCESS: OrderCode=%d, Amount=%d VND. [TODO: Write to Google Sheets]", orderCode, webhookData.Amount)
	wc.Store.UpdateStatus(orderCode, store.StatusPaid)

	w.WriteHeader(http.StatusOK)
}
