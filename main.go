package main

import (
	"html/template"
	"log"
	"net/http"
	"payos-demo/controllers"
	"payos-demo/store"
)

func main() {
	// Parse all templates once at startup.
	tmpl := template.Must(template.ParseGlob("templates/*.html"))

	// Instantiate the shared store. The eviction goroutine starts here.
	paymentStore := store.NewPaymentStore()

	// Wire up controllers with Dependency Injection.
	// Controllers receive the store — no global state.
	orderCtrl := &controllers.OrderController{
		Store:     paymentStore,
		Templates: tmpl,
	}
	webhookCtrl := &controllers.WebhookController{
		Store:     paymentStore,
		Templates: tmpl,
	}

	// Static file server
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Page routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})
	http.HandleFunc("/cancel/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "cancel.html", nil)
	})
	http.HandleFunc("/success/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "success.html", nil)
	})

	// Payment flow routes
	http.HandleFunc("/create-payment-link", orderCtrl.CreatePaymentLink)
	http.HandleFunc("/api/check-status", orderCtrl.CheckPaymentStatus)

	// Utility routes
	http.HandleFunc("/payment-link-info", controllers.GetPaymentLinkInfo)
	http.HandleFunc("/cancel-payment-link", controllers.CancelPaymentLink)

	// Webhook — receives verified payment events from PayOS.
	http.HandleFunc("/payos-webhook", webhookCtrl.VerifyPaymentWebhookData)

	// One-time admin route to register your webhook URL with PayOS.
	// Usage: GET /admin/confirm-webhook?url=https://your-domain.com/payos-webhook
	// Call this once after deployment. Delete or protect this route in production.
	http.HandleFunc("/admin/confirm-webhook", controllers.ConfirmWebhook)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
