# Architecture Overview

## Project Information
**Project Name**: payos-demo-golang
**Owner**: Vincent
**Created**: 2026-05-14
**Last Updated**: 2026-05-14

## High-Level Architecture
This is a lightweight Go web application that demonstrates integration with the payOS payment gateway. It operates using standard Go libraries (`net/http` and `html/template`), avoiding heavy web frameworks. It acts as an intermediary, generating payment links via the payOS SDK and redirecting clients to the payOS hosted checkout page. After checkout, users are redirected back to success or cancellation pages on this server.

## Key Architectural Decisions
- **Frameworkless HTTP**: Uses Go's native `net/http` package for routing and server functionality.
- **Server-Side Rendering (SSR)**: Simple HTML pages are served using `html/template` rather than an SPA framework for the frontend.
- **SDK Integration**: Utilizes the official `github.com/payOSHQ/payos-lib-golang` SDK to interact with payOS APIs, abstracting away raw HTTP calls and HMAC signature generation.
- **Environment Driven Configuration**: Secrets and domain information are injected via environment variables using `godotenv`.

## File Structure
- `/` (`main.go`): Application entry point and router definition.
- `/config/`: Configuration and environment variable loading (`constance.go`).
- `/controllers/`: Business logic handlers for payment operations (`order-controller.go`, `payment-controller.go`).
- `/templates/`: HTML templates for UI (`index`, `success`, `cancel`).
- `/static/`: Static assets (images, CSS, JS - if any).

## Constraints
- Does not currently persist order information in a database.
- Lacks a robust error handling middleware layer.
- Webhook routes are defined in controllers but not yet registered in the main router.
