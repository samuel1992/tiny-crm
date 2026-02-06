package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var repo *Repository
var PORT = "8080"

func setupRoutes(testing bool) *http.ServeMux {
	mux := http.NewServeMux()

	// Serve index.html at root path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "templates/index.html")
		}
	})

	// Protected API routes
	mux.HandleFunc("GET /api/companies", basicAuthMiddleware(getCompanies, testing))
	mux.HandleFunc("POST /api/companies", basicAuthMiddleware(createCompany, testing))
	mux.HandleFunc("GET /api/companies/{companyId}", basicAuthMiddleware(getCompany, testing))
	mux.HandleFunc("PUT /api/companies/{companyId}", basicAuthMiddleware(updateCompany, testing))
	mux.HandleFunc("DELETE /api/companies/{companyId}", basicAuthMiddleware(deleteCompany, testing))

	mux.HandleFunc("GET /api/remit", basicAuthMiddleware(getRemitInformations, testing))
	mux.HandleFunc("POST /api/remit", basicAuthMiddleware(createRemitInformation, testing))
	mux.HandleFunc("GET /api/remit/{remitId}", basicAuthMiddleware(getRemitInformation, testing))
	mux.HandleFunc("PUT /api/remit/{remitId}", basicAuthMiddleware(updateRemitInformation, testing))
	mux.HandleFunc("DELETE /api/remit/{remitId}", basicAuthMiddleware(deleteRemitInformation, testing))

	mux.HandleFunc("GET /api/products", basicAuthMiddleware(getProducts, testing))
	mux.HandleFunc("POST /api/products", basicAuthMiddleware(createProduct, testing))
	mux.HandleFunc("GET /api/products/{productId}", basicAuthMiddleware(getProduct, testing))
	mux.HandleFunc("PUT /api/products/{productId}", basicAuthMiddleware(updateProduct, testing))
	mux.HandleFunc("DELETE /api/products/{productId}", basicAuthMiddleware(deleteProduct, testing))

	mux.HandleFunc("GET /api/invoices", basicAuthMiddleware(getInvoices, testing))
	mux.HandleFunc("POST /api/invoices", basicAuthMiddleware(createInvoice, testing))
	mux.HandleFunc("GET /api/invoices/{invoiceId}", basicAuthMiddleware(getInvoice, testing))
	mux.HandleFunc("PUT /api/invoices/{invoiceId}", basicAuthMiddleware(updateInvoice, testing))
	mux.HandleFunc("DELETE /api/invoices/{invoiceId}", basicAuthMiddleware(deleteInvoice, testing))
	mux.HandleFunc("GET /api/invoices/{invoiceId}/open", basicAuthMiddleware(openInvoice, testing))
	mux.HandleFunc("GET /api/list_invoice_templates", basicAuthMiddleware(listTemplates, testing))
	mux.HandleFunc("POST /api/logout", logout)

	return mux
}

func main() {
	var err error
	repo, err = NewRepository()
	if err != nil {
		panic(err)
	}
	repo.Migrate()

	if len(os.Args) >= 2 && os.Args[1] == "--port" {
		PORT = os.Args[2]
	}

	// Handle CLI commands
	if len(os.Args) >= 2 && os.Args[1] == "adduser" {
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run . adduser <username> <password>")
			os.Exit(1)
		}

		username := os.Args[2]
		password := os.Args[3]

		// Check if user already exists
		existingUser, _ := repo.GetUserByUsername(username)
		if existingUser != nil {
			fmt.Printf("User '%s' already exists\n", username)
			os.Exit(1)
		}

		// Hash password
		hashedPassword, err := hashPassword(password)
		if err != nil {
			fmt.Printf("Error hashing password: %v\n", err)
			os.Exit(1)
		}

		// Create user
		user := &User{
			Username:     username,
			PasswordHash: hashedPassword,
		}

		if err := repo.CreateUser(user); err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User '%s' created successfully\n", username)
		return
	}

	mux := setupRoutes(false)

	fmt.Println("Running on port " + PORT)
	http.ListenAndServe(":"+PORT, mux)
}

func getCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := repo.GetCompanies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companies)
}

func createCompany(w http.ResponseWriter, r *http.Request) {
	var company Company
	if err := json.NewDecoder(r.Body).Decode(&company); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := repo.CreateCompany(&company); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(company)
}

func getCompany(w http.ResponseWriter, r *http.Request) {
	companyIdStr := r.PathValue("companyId")
	companyId, err := strconv.ParseUint(companyIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	company, err := repo.GetCompany(uint(companyId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func updateCompany(w http.ResponseWriter, r *http.Request) {
	companyIdStr := r.PathValue("companyId")
	companyId, err := strconv.ParseUint(companyIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	var company Company
	if err := json.NewDecoder(r.Body).Decode(&company); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	company.ID = uint(companyId)
	if err := repo.UpdateCompany(&company); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func deleteCompany(w http.ResponseWriter, r *http.Request) {
	companyIdStr := r.PathValue("companyId")
	companyId, err := strconv.ParseUint(companyIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	if err := repo.DeleteCompany(uint(companyId)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemitInformation handlers
func getRemitInformations(w http.ResponseWriter, r *http.Request) {
	remits, err := repo.GetRemitInformations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(remits)
}

func createRemitInformation(w http.ResponseWriter, r *http.Request) {
	var remit RemitInformation
	if err := json.NewDecoder(r.Body).Decode(&remit); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := repo.CreateRemitInformation(&remit); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(remit)
}

func getRemitInformation(w http.ResponseWriter, r *http.Request) {
	remitIdStr := r.PathValue("remitId")
	remitId, err := strconv.ParseUint(remitIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid remit ID", http.StatusBadRequest)
		return
	}

	remit, err := repo.GetRemitInformation(uint(remitId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(remit)
}

func updateRemitInformation(w http.ResponseWriter, r *http.Request) {
	remitIdStr := r.PathValue("remitId")
	remitId, err := strconv.ParseUint(remitIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid remit ID", http.StatusBadRequest)
		return
	}

	var remit RemitInformation
	if err := json.NewDecoder(r.Body).Decode(&remit); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	remit.ID = uint(remitId)
	if err := repo.UpdateRemitInformation(&remit); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(remit)
}

func deleteRemitInformation(w http.ResponseWriter, r *http.Request) {
	remitIdStr := r.PathValue("remitId")
	remitId, err := strconv.ParseUint(remitIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid remit ID", http.StatusBadRequest)
		return
	}

	if err := repo.DeleteRemitInformation(uint(remitId)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Product handlers
func getProducts(w http.ResponseWriter, r *http.Request) {
	products, err := repo.GetProducts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func createProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := repo.CreateProduct(&product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	productIdStr := r.PathValue("productId")
	productId, err := strconv.ParseUint(productIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := repo.GetProduct(uint(productId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func updateProduct(w http.ResponseWriter, r *http.Request) {
	productIdStr := r.PathValue("productId")
	productId, err := strconv.ParseUint(productIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	product.ID = uint(productId)
	if err := repo.UpdateProduct(&product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	productIdStr := r.PathValue("productId")
	productId, err := strconv.ParseUint(productIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := repo.DeleteProduct(uint(productId)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Invoice handlers
func getInvoices(w http.ResponseWriter, r *http.Request) {
	invoices, err := repo.GetInvoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoices)
}

func createInvoice(w http.ResponseWriter, r *http.Request) {
	var invoice Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := repo.CreateInvoice(&invoice); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the created invoice with all preloaded relationships
	createdInvoice, err := repo.GetInvoice(invoice.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdInvoice)
}

func getInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIdStr := r.PathValue("invoiceId")
	invoiceId, err := strconv.ParseUint(invoiceIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	invoice, err := repo.GetInvoice(uint(invoiceId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoice)
}

func updateInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIdStr := r.PathValue("invoiceId")
	invoiceId, err := strconv.ParseUint(invoiceIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	var invoice Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invoice.ID = uint(invoiceId)
	if err := repo.UpdateInvoice(&invoice); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the updated invoice with all preloaded relationships
	updatedInvoice, err := repo.GetInvoice(uint(invoiceId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedInvoice)
}

func deleteInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIdStr := r.PathValue("invoiceId")
	invoiceId, err := strconv.ParseUint(invoiceIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	if err := repo.DeleteInvoice(uint(invoiceId)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func listTemplates(w http.ResponseWriter, r *http.Request) {
	dirs, err := os.ReadDir("templates/invoices")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var templates []string
	for _, dir := range dirs {
		if !dir.IsDir() {
			templates = append(templates, dir.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func openInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIdStr := r.PathValue("invoiceId")
	invoiceId, err := strconv.ParseUint(invoiceIdStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid invoice ID", http.StatusBadRequest)
		return
	}

	templateName := r.URL.Query().Get("template")
	if templateName == "" {
		http.Error(w, "template query parameter is required", http.StatusBadRequest)
		return
	}

	invoice, err := repo.GetInvoice(uint(invoiceId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	templateData := struct {
		Invoice *Invoice
	}{
		Invoice: invoice,
	}

	tmplPath := filepath.Join("templates", "invoices", templateName)
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Error parsing template %s: %v", tmplPath, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, templateData)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmplPath, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	// Set WWW-Authenticate header to prompt for new credentials
	w.Header().Set("WWW-Authenticate", `Basic realm="Tiny CRM"`)
	http.Error(w, "Logged out successfully", http.StatusUnauthorized)
}
