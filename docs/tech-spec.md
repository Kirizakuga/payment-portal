# Technical Specifications

## Dependencies & Versions
- **Go**: 1.21.5
- **payOS SDK**: `github.com/payOSHQ/payos-lib-golang` v1.0.2
- **Environment Loader**: `github.com/joho/godotenv` v1.5.1

## Environment Variables
- `PAYOS_CLIENT_ID`: The Client ID provided by payOS.
- `PAYOS_API_KEY`: The API Key provided by payOS.
- `PAYOS_CHECKSUM_KEY`: The Checksum Key provided by payOS for webhook signature validation.

## Database Schema
N/A - This project currently operates without a database, relying solely on payOS for transaction state.

## Server Actions / Endpoints
### HTTP Routes
- `GET /`: Renders the index checkout page.
- `GET /cancel/`: Renders the cancellation page.
- `GET /success/`: Renders the success page.
- `POST /create-payment-link`: Generates a payOS checkout URL and redirects the user.
- `GET /payment-link-info`: Retrieves the status of a specific order via `orderId` query parameter.
- `POST /cancel-payment-link`: Cancels a specific order via `orderId` query parameter.

## Last Updated
2026-05-14
