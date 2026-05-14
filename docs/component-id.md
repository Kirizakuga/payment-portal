# CSS Component Mapping

This document maps the CSS classes and IDs defined in `static/stylesheet/style.css` to their respective components in the HTML templates (`templates/`).

## CSS Classes

*   **.main-box**
    *   **Usage:** Used in `index.html`, `cancel.html`, and `success.html`.
    *   **Component:** The main layout wrapper `div` that centers the content vertically and horizontally on the screen using Flexbox.

*   **.checkout**
    *   **Usage:** Used in `index.html`.
    *   **Component:** The container `div` holding the product information and the payment form. It provides the bordered card look.

*   **.product**
    *   **Usage:** Used in `index.html`.
    *   **Component:** The container `div` wrapping the product details (Name, Price, Quantity) to apply specific padding.

*   **.payment-titlte**
    *   **Usage:** Used in `cancel.html` and `success.html`.
    *   **Component:** The `h4` heading element displaying the transaction outcome (e.g., "Thanh toán thành công" or "Thanh toán thất bại"). Note: There is a typo in the class name ("titlte").

## CSS IDs

*   **#create-payment-link-btn**
    *   **Usage:** Used in `index.html`.
    *   **Component:** The submit `<button>` inside the `<form>`.
    *   **Action:** Fires a **POST** request to `/create-payment-link`.
    *   **Function:** Triggers `controllers.CreatePaymentLink` in Go, which uses the payOS SDK to generate a checkout URL and redirects the user.

*   **#return-page-btn**
    *   **Usage:** Used in `cancel.html` and `success.html`.
    *   **Component:** The anchor `<a>` tag styled as a button.
    *   **Action:** Fires a **GET** request to `/`.
    *   **Function:** Renders the main `index` template.

*   **#result**
    *   **Usage:** Currently unused in the provided HTML templates.
    *   **Component:** Styles a data table (with striped rows and hover effects), likely intended for displaying transaction results or order details dynamically.

*   **#query-string-table**
    *   **Usage:** Currently unused in the provided HTML templates.
    *   **Component:** Styles a table to be 50% width.

