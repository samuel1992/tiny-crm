package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

var repo *Repository
var tmpl *template.Template

type DashboardData struct {
	Companies  []Company
	Products   []Product
	RemitInfos []RemitInformation
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("templates/"))
	mux.Handle("/", fs)

	mux.HandleFunc("GET /company/", getCompanies)
	mux.HandleFunc("POST /company/", createCompany)
	mux.HandleFunc("GET /company/{companyId}", getCompany)
	mux.HandleFunc("PUT /company/{companyId}", updateCompany)
	mux.HandleFunc("DELETE /company/{companyId}", deleteCompany)

	mux.HandleFunc("GET /remit/", getRemitInformations)
	mux.HandleFunc("POST /remit/", createRemitInformation)
	mux.HandleFunc("GET /remit/{remitId}", getRemitInformation)
	mux.HandleFunc("PUT /remit/{remitId}", updateRemitInformation)
	mux.HandleFunc("DELETE /remit/{remitId}", deleteRemitInformation)

	mux.HandleFunc("GET /product/", getProducts)
	mux.HandleFunc("POST /product/", createProduct)
	mux.HandleFunc("GET /product/{productId}", getProduct)
	mux.HandleFunc("PUT /product/{productId}", updateProduct)
	mux.HandleFunc("DELETE /product/{productId}", deleteProduct)

	mux.HandleFunc("GET /invoice/", getInvoices)
	mux.HandleFunc("POST /invoice/", createInvoice)
	mux.HandleFunc("GET /invoice/{invoiceId}", getInvoice)
	mux.HandleFunc("PUT /invoice/{invoiceId}", updateInvoice)
	mux.HandleFunc("DELETE /invoice/{invoiceId}", deleteInvoice)

	return mux
}

func main() {
	var err error
	repo, err = NewRepository()
	if err != nil {
		panic(err)
	}
	repo.Migrate()

	// Parse templates
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		panic(err)
	}

	// Debug: Print available template names
	fmt.Println("Available templates:")
	for _, t := range tmpl.Templates() {
		fmt.Printf("- %s\n", t.Name())
	}

	mux := setupRoutes()

	fmt.Println("Running on port 8080")
	http.ListenAndServe(":8080", mux)
}

func getCompanies(w http.ResponseWriter, r *http.Request) {
	var companies []Company
	var err error
	companies, err = repo.GetCompanies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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

	// For HTMX requests, return empty response to remove the element
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
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

	// For HTMX requests, return empty response to remove the element
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
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

	// For HTMX requests, return empty response to remove the element
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invoice)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoice)
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
