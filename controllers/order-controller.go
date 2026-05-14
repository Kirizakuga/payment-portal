package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"payos-demo/config"
	"payos-demo/store"
	"strconv"
	"time"

	"github.com/payOSHQ/payos-lib-golang"
)

func init() {
	payos.Key(config.PAYOS_CLIENT_ID, config.PAYOS_API_KEY, config.PAYOS_CHECKSUM_KEY)
}

// OrderController holds a reference to the PaymentStore (Dependency Injection).
// This avoids global state, makes testing straightforward, and prevents race conditions.
type OrderController struct {
	Store     *store.PaymentStore
	Templates *template.Template
}

// QRPageData is the data contract between Go and the qr-display.html template.
type QRPageData struct {
	QRText    string
	Amount    int
	OrderCode int64
}

// CreatePaymentLink calls the PayOS API, stores the session, and renders the QR page.
// It no longer issues an HTTP redirect — the webhook is the source of truth for payment.
func (oc *OrderController) CreatePaymentLink(w http.ResponseWriter, r *http.Request) {
	// CRITICAL: The payos-lib-golang has a bug where it unmarshals numbers into float64
	// and then uses fmt.Sprintf("%v") to build the signature string.
	// If the number is >= 1,000,000, Go formats it in scientific notation (e.g., 1e+06),
	// which breaks the HMAC signature mismatching what PayOS sent.
	// We MUST keep the orderCode to 6 digits (max 999,999) to avoid this.
	orderCode := int64(time.Now().UnixMilli() % 1_000_000)

	body := payos.CheckoutRequestType{
		OrderCode:   int(orderCode),
		Amount:      2000,
		Description: "Thanh toan don hang",
		Items: []payos.Item{
			{
				Name:     "My tom Hao Hao ly",
				Price:    2000,
				Quantity: 1,
			},
		},
		CancelUrl: config.YOUR_DOMAIN + "/cancel/",
		ReturnUrl: config.YOUR_DOMAIN + "/success/",
	}

	data, err := payos.CreatePaymentLink(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Register the order in the store with PENDING status and the expected amount.
	// This must happen before rendering so the webhook handler can find the session.
	oc.Store.Set(orderCode, body.Amount)

	pageData := QRPageData{
		QRText:    data.QRCode,
		Amount:    body.Amount,
		OrderCode: orderCode,
	}

	err = oc.Templates.ExecuteTemplate(w, "qr-display.html", pageData)
	if err != nil {
		log.Printf("ERROR: failed to render qr-display.html: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// CheckPaymentStatus is polled by the frontend every 3 seconds.
// It returns the current status of an order as JSON.
//
// SMART POLLING: If the local store is PENDING (common on localhost where webhooks
// cannot reach), it will fallback to asking the PayOS API directly.
func (oc *OrderController) CheckPaymentStatus(w http.ResponseWriter, r *http.Request) {
	orderIdStr := r.URL.Query().Get("orderId")
	orderCode, err := strconv.ParseInt(orderIdStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid orderId", http.StatusBadRequest)
		return
	}

	session, ok := oc.Store.Get(orderCode)
	status := store.StatusPending
	if ok {
		status = session.Status
	}

	// If local store says PENDING, check with PayOS API directly.
	// We use a direct HTTP call here to bypass a bug in the payos-lib-golang
	// where GetPaymentLinkInformation fails signature verification on responses
	// containing transaction arrays.
	if status == store.StatusPending {
		client := &http.Client{}
		url := fmt.Sprintf("%s/v2/payment-requests/%s", config.PAYOS_BASE_URL, orderIdStr)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("x-client-id", config.PAYOS_CLIENT_ID)
		req.Header.Set("x-api-key", config.PAYOS_API_KEY)

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				Code string `json:"code"`
				Data struct {
					Status string `json:"status"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if result.Code == "00" && result.Data.Status == "PAID" {
					oc.Store.UpdateStatus(orderCode, store.StatusPaid)
					status = store.StatusPaid
					log.Printf("SMART POLLING SUCCESS: Order %d marked as PAID (Direct API)", orderCode)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": fmt.Sprintf("%s", status),
	})
}

// GetPaymentLinkInfo retrieves raw info from PayOS for debugging.
func GetPaymentLinkInfo(w http.ResponseWriter, r *http.Request) {
	orderId := r.URL.Query().Get("orderId")
	data, err := payos.GetPaymentLinkInformation(orderId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("PayOS status for %s: %s", orderId, data.Status)
}

// CancelPaymentLink cancels a PayOS payment link.
func CancelPaymentLink(w http.ResponseWriter, r *http.Request) {
	orderId := r.URL.Query().Get("orderId")
	data, err := payos.CancelPaymentLink(orderId, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Cancelled PayOS link for %s: %s", orderId, data.Status)
}

// ConfirmWebhook registers a webhook URL with PayOS.
func ConfirmWebhook(w http.ResponseWriter, r *http.Request) {
	webhookUrl := r.URL.Query().Get("url")
	webhookUrlRes, err := payos.ConfirmWebhook(webhookUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Print(webhookUrlRes)
}
