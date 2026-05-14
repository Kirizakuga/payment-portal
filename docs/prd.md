# Product Requirements Document

## Project Overview
**Project Name**: payos-demo-golang
**Owner**: Kiri-Kou
**Status**: 🟡 In Development

## Problem Statement
Merchants need a simple, reliable way to integrate Vietnamese payment methods into their applications. This project serves as a proof-of-concept and technical demonstration for processing payments using the payOS platform within a Golang environment.

## Features
- **Payment Link Generation**: Ability to dynamically create a checkout session for an item (e.g., "Mỳ tôm Hảo Hảo").
- **Payment Status Inquiry**: Functionality to look up the status of an existing order.
- **Order Cancellation**: Functionality to cancel an order before it is paid.
- **Webhook Verification (Pending Routing)**: Logic to receive and verify incoming Instant Payment Notifications (IPN) from payOS.

## Success Metrics
- Successfully generate a payment link and redirect to the payOS portal.
- Correctly capture the return redirect (Success/Cancel) after user interaction.
- Properly validate incoming webhook signatures to prevent spoofing.
