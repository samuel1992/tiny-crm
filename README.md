# Tiny CRM

A simple CRM system built with Go that manages companies, products, remit information, and invoices with customizable HTML templates.

## How to Run

### Prerequisites
- Go 1.24.2 or later
- SQLite database (created automatically)

### Setup
1. Create a user account:
```bash
go run . adduser <username> <password>
```

2. Start the server:
```bash
# Default port 8080
go run .

# Custom port
go run . --port 9090
```

3. Access the application:
- Web interface: http://localhost:8080
- API endpoints: `/api/*` (requires basic authentication)

## How to Build

### Local Development
```bash
go build -o tinycrm
./tinycrm
```

### Linux Deployment
Use the provided Docker build script for cross-compilation:
```bash
./build-linux.sh
```

This creates `tinycrm-linux` binary compatible with most Linux distributions.

## How to Add a New Invoice Template

### 1. Template Location
Create your template file in the `templates/invoices/` directory:
```
templates/invoices/your_template_name.html
```

### 2. Template Structure
Templates use Go's HTML template syntax with access to invoice data:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Invoice.Repr}}</title>
</head>
<body>
    <h1>Invoice: {{.Invoice.Number}}</h1>
    <p>Date: {{.Invoice.Date}}</p>
    <p>Company: {{.Invoice.Company.Name}}</p>
    
    <!-- Invoice items -->
    {{range .Invoice.InvoiceItems}}
    <div>
        <p>Product: {{.Product.Name}}</p>
        <p>Quantity: {{.Quantity}}</p>
        <p>Price: {{.UnitPrice}}</p>
    </div>
    {{end}}
    
    <p>Total: {{.Invoice.Total}}</p>
</body>
</html>
```

### 3. Available Data
Your template has access to the complete invoice object. Look the struct in `repository.go`.

### 4. Using the Template
Once created, your template will be automatically available via the API:
- List templates: `GET /api/list_invoice_templates`
- Generate invoice: `GET /api/invoices/{id}/open?template=your_template_name.html`

### Example Templates
See existing templates in `templates/invoices/` for reference:
- `default_invoice.html` - Basic invoice layout
- `concentrix_invoice.html` - Company-specific template
- `truelogic_invoice.html` - Another company template
