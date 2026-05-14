package store

import (
	"sync"
	"time"
)

// PaymentStatus represents the lifecycle state of a payment session.
type PaymentStatus string

const (
	StatusPending      PaymentStatus = "PENDING"
	StatusPaid         PaymentStatus = "PAID"
	StatusFraud        PaymentStatus = "PARTIAL_PAYMENT_FRAUD"
	// TTL for unpaid sessions — matches the standard VietQR transaction lifespan.
	sessionTTL = 15 * time.Minute
)

// PaymentSession holds everything we need to know about an in-flight order.
type PaymentSession struct {
	Status         PaymentStatus
	CreatedAt      time.Time
	ExpectedAmount int
	MSSV           string
}

// PaymentStore is a thread-safe, TTL-managed in-memory store.
// Keyed by int64 to safely accommodate UnixMilli (13-digit) order codes
// without overflowing on 32-bit systems or hitting JS MAX_SAFE_INTEGER.
type PaymentStore struct {
	mu       sync.Mutex
	sessions map[int64]PaymentSession
}

// NewPaymentStore creates a store and starts a background goroutine that
// evicts expired sessions every minute to prevent OOM from abandoned carts.
func NewPaymentStore() *PaymentStore {
	ps := &PaymentStore{
		sessions: make(map[int64]PaymentSession),
	}
	go ps.evictLoop()
	return ps
}

// Set registers a new order in the store with PENDING status.
func (ps *PaymentStore) Set(orderCode int64, expectedAmount int, mssv string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.sessions[orderCode] = PaymentSession{
		Status:         StatusPending,
		CreatedAt:      time.Now(),
		ExpectedAmount: expectedAmount,
		MSSV:           mssv,
	}
}

// Get retrieves a session. Returns the session and a boolean indicating existence.
func (ps *PaymentStore) Get(orderCode int64) (PaymentSession, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	session, ok := ps.sessions[orderCode]
	return session, ok
}

// UpdateStatus transitions an order to a new status (e.g., PAID or PARTIAL_PAYMENT_FRAUD).
func (ps *PaymentStore) UpdateStatus(orderCode int64, status PaymentStatus) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if session, ok := ps.sessions[orderCode]; ok {
		session.Status = status
		ps.sessions[orderCode] = session
	}
}

// evictLoop runs every minute and deletes PENDING sessions older than sessionTTL.
// PAID/FRAUD sessions are kept until they naturally expire too, for audit purposes.
func (ps *PaymentStore) evictLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ps.mu.Lock()
		for code, session := range ps.sessions {
			if time.Since(session.CreatedAt) > sessionTTL {
				delete(ps.sessions, code)
			}
		}
		ps.mu.Unlock()
	}
}
