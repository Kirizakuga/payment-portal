package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"payos-demo/config"
	"payos-demo/lib/gsheets"
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

type IndexPageData struct {
	Error string
}

// QRPageData is the data contract between Go and the qr-display.html template.
type QRPageData struct {
	QRText    string
	Amount    int
	OrderCode int64
}

// Index renders the home page with optional error message.
func (oc *OrderController) Index(w http.ResponseWriter, r *http.Request) {
	errormsg := r.URL.Query().Get("error")
	data := IndexPageData{Error: errormsg}
	oc.Templates.ExecuteTemplate(w, "index.html", data)
}

// CreatePaymentLink calls the PayOS API, stores the session, and renders the QR page.
// It no longer issues an HTTP redirect — the webhook is the source of truth for payment.
func (oc *OrderController) CreatePaymentLink(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	mssv := r.FormValue("mssv")

	// Validate MSSV against Google Sheets before creating payment
	exists, err := gsheets.ExistsMSSV(config.SPREADSHEET_ID, mssv)
	if err != nil {
		log.Printf("ERROR: Failed to validate MSSV %s: %v", mssv, err)
		http.Redirect(w, r, "/?error=System+error.+Please+try+again+later.", http.StatusSeeOther)
		return
	}
	if !exists {
		http.Redirect(w, r, "/?error=Student+ID+not+found+in+the+list.+Please+check+again.", http.StatusSeeOther)
		return
	}

	amount := 2000

	// CRITICAL: The payos-lib-golang has a bug where it unmarshals numbers into float64
	// and then uses fmt.Sprintf("%v") to build the signature string.
	// If the number is >= 1,000,000, Go formats it in scientific notation (e.g., 1e+06),
	// which breaks the HMAC signature mismatching what PayOS sent.
	// We MUST keep the orderCode to 6 digits (max 999,999) to avoid this.
	orderCode := int64(time.Now().UnixMilli() % 1_000_000)

	body := payos.CheckoutRequestType{
		OrderCode:   int(orderCode),
		Amount:      amount,
		Description: fmt.Sprintf("%s - Quỹ ITMC", mssv),
		Items: []payos.Item{
			{
				Name:     "Quỹ CLB ITMC",
				Price:    amount,
				Quantity: 1,
			},
		},
		CancelUrl: config.YOUR_DOMAIN + "/cancel/",
		ReturnUrl: config.YOUR_DOMAIN + "/success/",
	}

	data, err := payos.CreatePaymentLink(body)
	if err != nil {
		log.Printf("ERROR: Failed to create payment link: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Register the order in the store with PENDING status, expected amount, and MSSV.
	oc.Store.Set(orderCode, body.Amount, mssv)

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

					// Update Google Sheets automation
					if ok && session.MSSV != "" {
						go func() {
							err := gsheets.MarkAsPaid(config.SPREADSHEET_ID, session.MSSV)
							if err != nil {
								log.Printf("ERROR: Failed to update Google Sheets for MSSV %s: %v", session.MSSV, err)
							}
						}()
					}
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
