# Webhook Documentation

## Overview
External integration endpoints for receiving Instant Payment Notifications (IPN) from payOS. 

*Note: The controller logic exists (`VerifyPaymentWebhookData`, `ConfirmWebhook`), but these endpoints are not yet registered in `main.go`.*

## Endpoints

### Verify Payment Webhook Data
**Method**: POST
**Target Function**: `controllers.VerifyPaymentWebhookData`

**Description**:
Receives payment status updates from payOS. Validates the payload signature using `PAYOS_CHECKSUM_KEY` to ensure authenticity before processing.

#### Expected Request Payload Structure (Managed by SDK)
```json
{
  "code": "00",
  "desc": "success",
  "data": {
    "orderCode": 123456,
    "amount": 2000,
    "description": "Thanh toan don hang",
    "accountNumber": "...",
    "reference": "...",
    "transactionDateTime": "...",
    "currency": "VND",
    "paymentLinkId": "...",
    "code": "00",
    "desc": "success",
    "counterAccountBankId": "...",
    "counterAccountBankName": "...",
    "counterAccountName": "...",
    "counterAccountNumber": "...",
    "virtualAccountName": "...",
    "virtualAccountNumber": "..."
  },
  "signature": "..."
}
```

### Confirm Webhook Setup
**Method**: POST
**Target Function**: `controllers.ConfirmWebhook`

**Description**:
Used during the initial setup phase with payOS to confirm the webhook URL is reachable and functioning correctly.
