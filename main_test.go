package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestServer(t *testing.T) (*httptest.Server, *Repository) {
	// Create in-memory database
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Enable foreign key constraints in SQLite
	testDB.Exec("PRAGMA foreign_keys = ON")

	// Create repository with test database
	testRepo, err := NewRepositoryWithDB(testDB)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Run migrations
	err = testDB.AutoMigrate(
		&RemitInformation{},
		&RemitInformationLine{},
		&Product{},
		&Company{},
		&Invoice{},
		&InvoiceLine{},
		&RecurringInvoice{},
		&RecurringInvoiceLine{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Store original repo and set test repo
	originalRepo := repo
	repo = testRepo

	// Use the same route setup as main.go
	mux := setupRoutes(true)
	server := httptest.NewServer(mux)

	// Clean up function to restore original repo
	t.Cleanup(func() {
		repo = originalRepo
		server.Close()
	})

	return server, testRepo
}

func makeRequest(server *httptest.Server, method, endpoint, body string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, server.URL+endpoint, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, responseBody, nil
}

func createTestData(testRepo *Repository) (companyID, productID, remitID uint, err error) {
	// Create test company
	company := Company{
		Name:     "Test Company Ltd",
		Document: "12.345.678/0001-90",
		Address:  "123 Test Street, Test City",
	}
	if err = testRepo.CreateCompany(&company); err != nil {
		return 0, 0, 0, err
	}
	companyID = company.ID

	// Create test product
	product := Product{
		Name:        "Test Product",
		Description: stringPtr("Test product description"),
		Price:       99.99,
	}
	if err = testRepo.CreateProduct(&product); err != nil {
		return 0, 0, 0, err
	}
	productID = product.ID

	// Create test remit information
	remit := RemitInformation{
		Name: "Test Remit Info",
		Lines: []RemitInformationLine{
			{Key: "bank", Value: "Test Bank"},
			{Key: "account", Value: "123456789"},
		},
	}
	if err = testRepo.CreateRemitInformation(&remit); err != nil {
		return 0, 0, 0, err
	}
	remitID = remit.ID

	return companyID, productID, remitID, nil
}

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}

// Company Tests
func TestCompanyCreate(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	companyJSON := `{
		"name": "Integration Test Company",
		"document": "98.765.432/0001-10",
		"address": "456 Integration Ave, Test City, ST"
	}`

	resp, body, err := makeRequest(server, "POST", "/api/companies", companyJSON)
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var createdCompany Company
	if err := json.Unmarshal(body, &createdCompany); err != nil {
		t.Fatalf("Failed to unmarshal created company: %v", err)
	}

	if createdCompany.ID == 0 {
		t.Error("Created company should have an ID")
	}
	if createdCompany.Name != "Integration Test Company" {
		t.Errorf("Expected name 'Integration Test Company', got '%s'", createdCompany.Name)
	}
	if createdCompany.Document != "98.765.432/0001-10" {
		t.Errorf("Expected document '98.765.432/0001-10', got '%s'", createdCompany.Document)
	}
}

func TestCompanyGet(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test company first
	company := Company{
		Name:     "Test Company",
		Document: "12.345.678/0001-90",
		Address:  "123 Test Street",
	}
	if err := testRepo.CreateCompany(&company); err != nil {
		t.Fatalf("Failed to create test company: %v", err)
	}

	resp, body, err := makeRequest(server, "GET", "/api/companies/"+strconv.Itoa(int(company.ID)), "")
	if err != nil {
		t.Fatalf("Failed to get company: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedCompany Company
	if err := json.Unmarshal(body, &retrievedCompany); err != nil {
		t.Fatalf("Failed to unmarshal retrieved company: %v", err)
	}

	if retrievedCompany.ID != company.ID {
		t.Errorf("Expected ID %d, got %d", company.ID, retrievedCompany.ID)
	}
	if retrievedCompany.Name != "Test Company" {
		t.Errorf("Expected name 'Test Company', got '%s'", retrievedCompany.Name)
	}
}

func TestCompanyList(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test companies
	companies := []Company{
		{Name: "Company 1", Document: "11.111.111/0001-11", Address: "Address 1"},
		{Name: "Company 2", Document: "22.222.222/0001-22", Address: "Address 2"},
	}

	for i := range companies {
		if err := testRepo.CreateCompany(&companies[i]); err != nil {
			t.Fatalf("Failed to create test company: %v", err)
		}
	}

	resp, body, err := makeRequest(server, "GET", "/api/companies", "")
	if err != nil {
		t.Fatalf("Failed to list companies: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedCompanies []Company
	if err := json.Unmarshal(body, &retrievedCompanies); err != nil {
		t.Fatalf("Failed to unmarshal companies list: %v", err)
	}

	if len(retrievedCompanies) < 2 {
		t.Errorf("Expected at least 2 companies, got %d", len(retrievedCompanies))
	}
}

func TestCompanyUpdate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test company first
	company := Company{
		Name:     "Original Company",
		Document: "12.345.678/0001-90",
		Address:  "Original Address",
	}
	if err := testRepo.CreateCompany(&company); err != nil {
		t.Fatalf("Failed to create test company: %v", err)
	}

	updateJSON := `{
		"name": "Updated Company Name",
		"document": "12.345.678/0001-90",
		"address": "789 Updated Street, New City, ST"
	}`

	resp, body, err := makeRequest(server, "PUT", "/api/companies/"+strconv.Itoa(int(company.ID)), updateJSON)
	if err != nil {
		t.Fatalf("Failed to update company: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var updatedCompany Company
	if err := json.Unmarshal(body, &updatedCompany); err != nil {
		t.Fatalf("Failed to unmarshal updated company: %v", err)
	}

	if updatedCompany.Name != "Updated Company Name" {
		t.Errorf("Expected updated name 'Updated Company Name', got '%s'", updatedCompany.Name)
	}
	if updatedCompany.Address != "789 Updated Street, New City, ST" {
		t.Errorf("Expected updated address, got '%s'", updatedCompany.Address)
	}
}

func TestCompanyDelete(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test company first
	company := Company{
		Name:     "Company to Delete",
		Document: "12.345.678/0001-90",
		Address:  "Delete Address",
	}
	if err := testRepo.CreateCompany(&company); err != nil {
		t.Fatalf("Failed to create test company: %v", err)
	}

	resp, _, err := makeRequest(server, "DELETE", "/api/companies/"+strconv.Itoa(int(company.ID)), "")
	if err != nil {
		t.Fatalf("Failed to delete company: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify deletion by trying to fetch
	resp, body, err := makeRequest(server, "GET", "/api/companies/"+strconv.Itoa(int(company.ID)), "")
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Verify company is removed from database
	var count int64
	testRepo.db.Model(&Company{}).Where("id = ?", company.ID).Count(&count)
	if count != 0 {
		t.Error("Company should be deleted from database")
	}
}

// Product Tests
func TestProductCreate(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	productJSON := `{
		"name": "Integration Test Product",
		"description": "A product for integration testing",
		"price": 149.99
	}`

	resp, body, err := makeRequest(server, "POST", "/api/products", productJSON)
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var createdProduct Product
	if err := json.Unmarshal(body, &createdProduct); err != nil {
		t.Fatalf("Failed to unmarshal created product: %v", err)
	}

	if createdProduct.ID == 0 {
		t.Error("Created product should have an ID")
	}
	if createdProduct.Name != "Integration Test Product" {
		t.Errorf("Expected name 'Integration Test Product', got '%s'", createdProduct.Name)
	}
	if createdProduct.Price != 149.99 {
		t.Errorf("Expected price 149.99, got %f", createdProduct.Price)
	}
	if createdProduct.Description == nil || *createdProduct.Description != "A product for integration testing" {
		t.Error("Expected description to be set correctly")
	}
}

func TestProductCreateWithoutDescription(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	productJSON := `{
		"name": "Product Without Description",
		"price": 99.99
	}`

	resp, body, err := makeRequest(server, "POST", "/api/products", productJSON)
	if err != nil {
		t.Fatalf("Failed to create product without description: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var productNoDesc Product
	if err := json.Unmarshal(body, &productNoDesc); err != nil {
		t.Fatalf("Failed to unmarshal product without description: %v", err)
	}

	if productNoDesc.Description != nil {
		t.Error("Description should be nil when not provided")
	}
	if productNoDesc.Name != "Product Without Description" {
		t.Errorf("Expected name 'Product Without Description', got '%s'", productNoDesc.Name)
	}
	if productNoDesc.Price != 99.99 {
		t.Errorf("Expected price 99.99, got %f", productNoDesc.Price)
	}
}

func TestProductGet(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test product first
	product := Product{
		Name:        "Test Product",
		Description: stringPtr("Test description"),
		Price:       149.99,
	}
	if err := testRepo.CreateProduct(&product); err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	resp, body, err := makeRequest(server, "GET", "/api/products/"+strconv.Itoa(int(product.ID)), "")
	if err != nil {
		t.Fatalf("Failed to get product: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedProduct Product
	if err := json.Unmarshal(body, &retrievedProduct); err != nil {
		t.Fatalf("Failed to unmarshal retrieved product: %v", err)
	}

	if retrievedProduct.ID != product.ID {
		t.Errorf("Expected ID %d, got %d", product.ID, retrievedProduct.ID)
	}
	if retrievedProduct.Price != 149.99 {
		t.Errorf("Expected price 149.99, got %f", retrievedProduct.Price)
	}
}

func TestProductList(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test products
	products := []Product{
		{Name: "Product 1", Price: 10.99},
		{Name: "Product 2", Description: stringPtr("Product 2 desc"), Price: 20.99},
	}

	for i := range products {
		if err := testRepo.CreateProduct(&products[i]); err != nil {
			t.Fatalf("Failed to create test product: %v", err)
		}
	}

	resp, body, err := makeRequest(server, "GET", "/api/products", "")
	if err != nil {
		t.Fatalf("Failed to list products: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedProducts []Product
	if err := json.Unmarshal(body, &retrievedProducts); err != nil {
		t.Fatalf("Failed to unmarshal products list: %v", err)
	}

	if len(retrievedProducts) < 2 {
		t.Errorf("Expected at least 2 products, got %d", len(retrievedProducts))
	}
}

func TestProductUpdate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test product first
	product := Product{
		Name:        "Original Product",
		Description: stringPtr("Original description"),
		Price:       149.99,
	}
	if err := testRepo.CreateProduct(&product); err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	updateJSON := `{
		"name": "Updated Product Name",
		"description": "Updated description for the product",
		"price": 199.99
	}`

	resp, body, err := makeRequest(server, "PUT", "/api/products/"+strconv.Itoa(int(product.ID)), updateJSON)
	if err != nil {
		t.Fatalf("Failed to update product: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var updatedProduct Product
	if err := json.Unmarshal(body, &updatedProduct); err != nil {
		t.Fatalf("Failed to unmarshal updated product: %v", err)
	}

	if updatedProduct.Name != "Updated Product Name" {
		t.Errorf("Expected updated name 'Updated Product Name', got '%s'", updatedProduct.Name)
	}
	if updatedProduct.Price != 199.99 {
		t.Errorf("Expected updated price 199.99, got %f", updatedProduct.Price)
	}
	if updatedProduct.Description == nil || *updatedProduct.Description != "Updated description for the product" {
		t.Error("Expected updated description")
	}
}

func TestProductDelete(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test product first
	product := Product{
		Name:  "Product to Delete",
		Price: 99.99,
	}
	if err := testRepo.CreateProduct(&product); err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	resp, _, err := makeRequest(server, "DELETE", "/api/products/"+strconv.Itoa(int(product.ID)), "")
	if err != nil {
		t.Fatalf("Failed to delete product: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify deletion by trying to fetch
	resp, body, err := makeRequest(server, "GET", "/api/products/"+strconv.Itoa(int(product.ID)), "")
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Verify product is removed from database
	var count int64
	testRepo.db.Model(&Product{}).Where("id = ?", product.ID).Count(&count)
	if count != 0 {
		t.Error("Product should be deleted from database")
	}
}

// RemitInformation Tests
func TestRemitInformationCreate(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	remitJSON := `{
		"name": "Test Bank Info",
		"lines": [
			{"key": "bank", "value": "Test Bank"},
			{"key": "account", "value": "123456789"},
			{"key": "agency", "value": "0001"}
		]
	}`

	resp, body, err := makeRequest(server, "POST", "/api/remit", remitJSON)
	if err != nil {
		t.Fatalf("Failed to create remit information: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var createdRemit RemitInformation
	if err := json.Unmarshal(body, &createdRemit); err != nil {
		t.Fatalf("Failed to unmarshal created remit: %v", err)
	}

	if createdRemit.ID == 0 {
		t.Error("Created remit should have an ID")
	}
	if createdRemit.Name != "Test Bank Info" {
		t.Errorf("Expected name 'Test Bank Info', got '%s'", createdRemit.Name)
	}
	if len(createdRemit.Lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(createdRemit.Lines))
	}
}

func TestRemitInformationGet(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test remit information first
	remit := RemitInformation{
		Name: "Test Remit Info",
		Lines: []RemitInformationLine{
			{Key: "bank", Value: "Test Bank"},
			{Key: "account", Value: "987654321"},
		},
	}
	if err := testRepo.CreateRemitInformation(&remit); err != nil {
		t.Fatalf("Failed to create test remit: %v", err)
	}

	resp, body, err := makeRequest(server, "GET", "/api/remit/"+strconv.Itoa(int(remit.ID)), "")
	if err != nil {
		t.Fatalf("Failed to get remit information: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedRemit RemitInformation
	if err := json.Unmarshal(body, &retrievedRemit); err != nil {
		t.Fatalf("Failed to unmarshal retrieved remit: %v", err)
	}

	if retrievedRemit.ID != remit.ID {
		t.Errorf("Expected ID %d, got %d", remit.ID, retrievedRemit.ID)
	}
	if retrievedRemit.Name != "Test Remit Info" {
		t.Errorf("Expected name 'Test Remit Info', got '%s'", retrievedRemit.Name)
	}
	if len(retrievedRemit.Lines) != 2 {
		t.Errorf("Expected 2 preloaded lines, got %d", len(retrievedRemit.Lines))
	}
}

func TestRemitInformationList(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test remit informations
	remits := []RemitInformation{
		{Name: "Bank 1", Lines: []RemitInformationLine{{Key: "bank", Value: "Bank 1"}}},
		{Name: "Bank 2", Lines: []RemitInformationLine{{Key: "bank", Value: "Bank 2"}}},
	}

	for i := range remits {
		if err := testRepo.CreateRemitInformation(&remits[i]); err != nil {
			t.Fatalf("Failed to create test remit: %v", err)
		}
	}

	resp, body, err := makeRequest(server, "GET", "/api/remit", "")
	if err != nil {
		t.Fatalf("Failed to list remit informations: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedRemits []RemitInformation
	if err := json.Unmarshal(body, &retrievedRemits); err != nil {
		t.Fatalf("Failed to unmarshal remit informations list: %v", err)
	}

	if len(retrievedRemits) < 2 {
		t.Errorf("Expected at least 2 remit informations, got %d", len(retrievedRemits))
	}

	// Verify lines are preloaded
	for _, remit := range retrievedRemits {
		if len(remit.Lines) == 0 {
			t.Error("Lines should be preloaded in list response")
		}
	}
}

func TestRemitInformationUpdate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test remit information first
	remit := RemitInformation{
		Name: "Original Bank",
		Lines: []RemitInformationLine{
			{Key: "bank", Value: "Original Bank"},
		},
	}
	if err := testRepo.CreateRemitInformation(&remit); err != nil {
		t.Fatalf("Failed to create test remit: %v", err)
	}

	updateJSON := `{
		"name": "Updated Bank Info",
		"lines": [
			{"key": "bank", "value": "Updated Bank"},
			{"key": "account", "value": "111222333"}
		]
	}`

	resp, body, err := makeRequest(server, "PUT", "/api/remit/"+strconv.Itoa(int(remit.ID)), updateJSON)
	if err != nil {
		t.Fatalf("Failed to update remit information: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var updatedRemit RemitInformation
	if err := json.Unmarshal(body, &updatedRemit); err != nil {
		t.Fatalf("Failed to unmarshal updated remit: %v", err)
	}

	if updatedRemit.Name != "Updated Bank Info" {
		t.Errorf("Expected updated name 'Updated Bank Info', got '%s'", updatedRemit.Name)
	}
}

func TestRemitInformationDelete(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create test remit information first
	remit := RemitInformation{
		Name: "Remit to Delete",
	}
	if err := testRepo.CreateRemitInformation(&remit); err != nil {
		t.Fatalf("Failed to create test remit: %v", err)
	}

	// Create lines separately to ensure foreign key is set
	line := RemitInformationLine{
		Key:                "bank",
		Value:              "Delete Bank",
		RemitInformationID: remit.ID,
	}
	if err := testRepo.db.Create(&line).Error; err != nil {
		t.Fatalf("Failed to create test remit line: %v", err)
	}

	// Verify lines were created with proper foreign key
	var initialLineCount int64
	testRepo.db.Model(&RemitInformationLine{}).Where("remit_information_id = ?", remit.ID).Count(&initialLineCount)
	if initialLineCount == 0 {
		t.Error("RemitInformationLines should be created with the remit")
	}

	resp, _, err := makeRequest(server, "DELETE", "/api/remit/"+strconv.Itoa(int(remit.ID)), "")
	if err != nil {
		t.Fatalf("Failed to delete remit information: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify deletion by trying to fetch
	resp, body, err := makeRequest(server, "GET", "/api/remit/"+strconv.Itoa(int(remit.ID)), "")
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Verify cascade deletion of lines - only check if lines were initially created
	if initialLineCount > 0 {
		var lineCount int64
		testRepo.db.Model(&RemitInformationLine{}).Where("remit_information_id = ?", remit.ID).Count(&lineCount)
		if lineCount != 0 {
			t.Error("RemitInformationLines should be cascade deleted")
		}
	}
}

// Invoice Tests
func TestInvoiceCreate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create prerequisite data
	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	invoiceJSON := fmt.Sprintf(`{
		"number": 1001,
		"additional_information": "Test invoice",
		"discount": 10.50,
		"penalty": 5.25,
		"due_date": "2024-12-31T23:59:59Z",
		"remit_information_id": %d,
		"company_id": %d,
		"client_id": %d,
		"invoice_lines": [
			{
				"product_id": %d,
				"quantity": 2,
				"description": "Test product line"
			}
		]
	}`, remitID, companyID, companyID, productID)

	resp, body, err := makeRequest(server, "POST", "/api/invoices", invoiceJSON)
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var createdInvoice Invoice
	if err := json.Unmarshal(body, &createdInvoice); err != nil {
		t.Fatalf("Failed to unmarshal created invoice: %v", err)
	}

	if createdInvoice.ID == 0 {
		t.Error("Created invoice should have an ID")
	}
	if createdInvoice.UUID == (uuid.UUID{}) {
		t.Error("Created invoice should have a UUID")
	}
	if createdInvoice.Number == nil || *createdInvoice.Number != 1001 {
		t.Error("Invoice number should be set correctly")
	}
}

func TestInvoiceGet(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create prerequisite data
	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create test invoice
	dueDate := time.Now().AddDate(0, 1, 0)
	invoice := Invoice{
		Number:                intPtr(2001),
		AdditionalInformation: stringPtr("Test invoice for get"),
		Discount:              15.00,
		Penalty:               0.00,
		DueDate:               dueDate,
		RemitInformationID:    remitID,
		CompanyID:             companyID,
		ClientID:              companyID,
		InvoiceLines: []InvoiceLine{
			{
				ProductID:   productID,
				Quantity:    3,
				Description: stringPtr("Get test line"),
			},
		},
	}
	if err := testRepo.CreateInvoice(&invoice); err != nil {
		t.Fatalf("Failed to create test invoice: %v", err)
	}

	resp, body, err := makeRequest(server, "GET", "/api/invoices/"+strconv.Itoa(int(invoice.ID)), "")
	if err != nil {
		t.Fatalf("Failed to get invoice: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedInvoice Invoice
	if err := json.Unmarshal(body, &retrievedInvoice); err != nil {
		t.Fatalf("Failed to unmarshal retrieved invoice: %v", err)
	}

	if retrievedInvoice.ID != invoice.ID {
		t.Errorf("Expected ID %d, got %d", invoice.ID, retrievedInvoice.ID)
	}
	// Verify relationships are preloaded
	if len(retrievedInvoice.InvoiceLines) == 0 {
		t.Error("InvoiceLines should be preloaded")
	}
	if retrievedInvoice.Company.ID == 0 {
		t.Error("Company should be preloaded")
	}
	if retrievedInvoice.Client.ID == 0 {
		t.Error("Client should be preloaded")
	}
	if retrievedInvoice.RemitInformation.ID == 0 {
		t.Error("RemitInformation should be preloaded")
	}
}

func TestInvoiceList(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create prerequisite data
	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create test invoices
	invoices := []Invoice{
		{
			Number:             intPtr(3001),
			Discount:           0.00,
			Penalty:            0.00,
			DueDate:            time.Now().AddDate(0, 1, 0),
			RemitInformationID: remitID,
			CompanyID:          companyID,
			ClientID:           companyID,
			InvoiceLines: []InvoiceLine{
				{ProductID: productID, Quantity: 1},
			},
		},
		{
			Number:             intPtr(3002),
			Discount:           5.00,
			Penalty:            0.00,
			DueDate:            time.Now().AddDate(0, 2, 0),
			RemitInformationID: remitID,
			CompanyID:          companyID,
			ClientID:           companyID,
			InvoiceLines: []InvoiceLine{
				{ProductID: productID, Quantity: 2},
			},
		},
	}

	for i := range invoices {
		if err := testRepo.CreateInvoice(&invoices[i]); err != nil {
			t.Fatalf("Failed to create test invoice: %v", err)
		}
	}

	resp, body, err := makeRequest(server, "GET", "/api/invoices", "")
	if err != nil {
		t.Fatalf("Failed to list invoices: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var retrievedInvoices []Invoice
	if err := json.Unmarshal(body, &retrievedInvoices); err != nil {
		t.Fatalf("Failed to unmarshal invoices list: %v", err)
	}

	if len(retrievedInvoices) < 2 {
		t.Errorf("Expected at least 2 invoices, got %d", len(retrievedInvoices))
	}

	// Verify all relationships are preloaded
	for _, invoice := range retrievedInvoices {
		if len(invoice.InvoiceLines) == 0 {
			t.Error("InvoiceLines should be preloaded in list response")
		}
		if invoice.Company.ID == 0 {
			t.Error("Company should be preloaded in list response")
		}
	}
}

func TestInvoiceUpdate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create prerequisite data
	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create test invoice
	dueDate := time.Now().AddDate(0, 1, 0)
	invoice := Invoice{
		Number:             intPtr(4001),
		Discount:           0.00,
		Penalty:            0.00,
		DueDate:            dueDate,
		RemitInformationID: remitID,
		CompanyID:          companyID,
		ClientID:           companyID,
		InvoiceLines: []InvoiceLine{
			{ProductID: productID, Quantity: 1},
		},
	}
	if err := testRepo.CreateInvoice(&invoice); err != nil {
		t.Fatalf("Failed to create test invoice: %v", err)
	}

	updateJSON := fmt.Sprintf(`{
		"number": 4002,
		"additional_information": "Updated invoice info",
		"discount": 25.00,
		"penalty": 10.00,
		"due_date": "2025-01-31T23:59:59Z",
		"remit_information_id": %d,
		"company_id": %d,
		"client_id": %d,
		"invoice_lines": [
			{
				"product_id": %d,
				"quantity": 5,
				"description": "Updated line"
			}
		]
	}`, remitID, companyID, companyID, productID)

	resp, body, err := makeRequest(server, "PUT", "/api/invoices/"+strconv.Itoa(int(invoice.ID)), updateJSON)
	if err != nil {
		t.Fatalf("Failed to update invoice: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var updatedInvoice Invoice
	if err := json.Unmarshal(body, &updatedInvoice); err != nil {
		t.Fatalf("Failed to unmarshal updated invoice: %v", err)
	}

	if updatedInvoice.Number == nil || *updatedInvoice.Number != 4002 {
		t.Error("Invoice number should be updated")
	}
	if updatedInvoice.Discount != 25.00 {
		t.Errorf("Expected discount 25.00, got %f", updatedInvoice.Discount)
	}
}

func TestInvoiceUpdatePreservesUUIDAndIssueDate(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	invoice := Invoice{
		Number:             intPtr(6001),
		DueDate:            time.Now().AddDate(0, 1, 0),
		RemitInformationID: remitID,
		CompanyID:          companyID,
		ClientID:           companyID,
		InvoiceLines: []InvoiceLine{
			{ProductID: productID, Quantity: 1},
		},
	}
	if err := testRepo.CreateInvoice(&invoice); err != nil {
		t.Fatalf("Failed to create test invoice: %v", err)
	}

	originalUUID := invoice.UUID
	originalIssueDate := invoice.IssueDate
	if originalUUID == (uuid.UUID{}) {
		t.Fatal("Precondition failed: created invoice should have a non-zero UUID")
	}
	if originalIssueDate.IsZero() {
		t.Fatal("Precondition failed: created invoice should have a non-zero IssueDate")
	}

	// Mimic the frontend: PUT body omits uuid and issue_date.
	updateJSON := fmt.Sprintf(`{
		"number": 6002,
		"discount": 0,
		"penalty": 0,
		"due_date": "2025-06-30T23:59:59Z",
		"remit_information_id": %d,
		"company_id": %d,
		"client_id": %d,
		"invoice_lines": [
			{"product_id": %d, "quantity": 2}
		]
	}`, remitID, companyID, companyID, productID)

	resp, body, err := makeRequest(server, "PUT", "/api/invoices/"+strconv.Itoa(int(invoice.ID)), updateJSON)
	if err != nil {
		t.Fatalf("Failed to update invoice: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
	}

	reloaded, err := testRepo.GetInvoice(invoice.ID)
	if err != nil {
		t.Fatalf("Failed to reload invoice: %v", err)
	}

	if reloaded.UUID != originalUUID {
		t.Errorf("UUID changed across update: was %s, now %s", originalUUID, reloaded.UUID)
	}
	if !reloaded.IssueDate.Equal(originalIssueDate) {
		t.Errorf("IssueDate changed across update: was %s, now %s", originalIssueDate, reloaded.IssueDate)
	}
}

func TestInvoiceDelete(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	// Create prerequisite data
	companyID, productID, remitID, err := createTestData(testRepo)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create test invoice
	dueDate := time.Now().AddDate(0, 1, 0)
	invoice := Invoice{
		Number:             intPtr(5001),
		DueDate:            dueDate,
		RemitInformationID: remitID,
		CompanyID:          companyID,
		ClientID:           companyID,
	}
	if err := testRepo.CreateInvoice(&invoice); err != nil {
		t.Fatalf("Failed to create test invoice: %v", err)
	}

	// Create invoice lines separately to ensure foreign key is set
	line := InvoiceLine{
		InvoiceID: invoice.ID,
		ProductID: productID,
		Quantity:  1,
	}
	if err := testRepo.db.Create(&line).Error; err != nil {
		t.Fatalf("Failed to create test invoice line: %v", err)
	}

	// Verify invoice lines were created with proper foreign key
	var initialLineCount int64
	testRepo.db.Model(&InvoiceLine{}).Where("invoice_id = ?", invoice.ID).Count(&initialLineCount)
	if initialLineCount == 0 {
		t.Error("InvoiceLines should be created with the invoice")
	}

	resp, _, err := makeRequest(server, "DELETE", "/api/invoices/"+strconv.Itoa(int(invoice.ID)), "")
	if err != nil {
		t.Fatalf("Failed to delete invoice: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify deletion by trying to fetch
	resp, body, err := makeRequest(server, "GET", "/api/invoices/"+strconv.Itoa(int(invoice.ID)), "")
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Verify cascade deletion of invoice lines - only check if lines were initially created
	if initialLineCount > 0 {
		var lineCount int64
		testRepo.db.Model(&InvoiceLine{}).Where("invoice_id = ?", invoice.ID).Count(&lineCount)
		if lineCount != 0 {
			t.Error("InvoiceLines should be cascade deleted")
		}
	}
}

// Error handling tests
func TestCompanyGetInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/companies/invalid", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestCompanyGetNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/companies/99999", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestCompanyCreateMalformedJSON(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "POST", "/api/companies", `{"name": "Test", invalid json}`)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestCompanyUpdateInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	companyData := `{"name": "Updated Company", "document": "987654321", "address": "Updated Address"}`
	resp, body, err := makeRequest(server, "PUT", "/api/companies/invalid", companyData)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestCompanyDeleteInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "DELETE", "/api/companies/invalid", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestProductGetInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/products/invalid", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestProductGetNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/products/99999", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestProductCreateMalformedJSON(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "POST", "/api/products", `{"name": "Test", invalid json}`)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestRemitInformationGetInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/remit/invalid", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestRemitInformationGetNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/remit/99999", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestInvoiceGetInvalidID(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/invoices/invalid", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestInvoiceGetNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "GET", "/api/invoices/99999", "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

func TestInvoiceCreateMalformedJSON(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	resp, body, err := makeRequest(server, "POST", "/api/invoices", `{"number": 1, invalid json}`)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Response: %s", resp.StatusCode, string(body))
	}
}

// --- Recurring invoices ---

func seedRecurringTemplate(t *testing.T, repo *Repository, generateOnDay int, rule string, dayOfMonth *int, nextMonth bool) (*RecurringInvoice, uint) {
	t.Helper()
	companyID, productID, remitID, err := createTestData(repo)
	if err != nil {
		t.Fatalf("seed prerequisites: %v", err)
	}
	ri := &RecurringInvoice{
		Name:               "Monthly Test",
		IsActive:           true,
		GenerateOnDay:      generateOnDay,
		DueDateRule:        rule,
		DueDayOfMonth:      dayOfMonth,
		DueDayInNextMonth:  nextMonth,
		CompanyID:          companyID,
		ClientID:           companyID,
		RemitInformationID: remitID,
		BaseDiscount:       0,
		BasePenalty:        0,
		Lines: []RecurringInvoiceLine{
			{ProductID: productID, Quantity: 2, Description: stringPtr("recurring line")},
		},
	}
	if err := repo.CreateRecurringInvoice(ri); err != nil {
		t.Fatalf("create recurring template: %v", err)
	}
	return ri, productID
}

func TestCalculateDueDate(t *testing.T) {
	loc := time.UTC

	cases := []struct {
		name      string
		issue     time.Time
		rule      string
		day       *int
		nextMonth bool
		want      time.Time
	}{
		{
			name:  "end of January",
			issue: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),
			rule:  DueRuleEndOfMonth,
			want:  time.Date(2026, 1, 31, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of February non-leap",
			issue: time.Date(2026, 2, 10, 0, 0, 0, 0, loc),
			rule:  DueRuleEndOfMonth,
			want:  time.Date(2026, 2, 28, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of December rolls to year end",
			issue: time.Date(2026, 12, 1, 0, 0, 0, 0, loc),
			rule:  DueRuleEndOfMonth,
			want:  time.Date(2026, 12, 31, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of next month from December",
			issue: time.Date(2026, 12, 15, 0, 0, 0, 0, loc),
			rule:  DueRuleEndOfNextMonth,
			want:  time.Date(2027, 1, 31, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of next month from May",
			issue: time.Date(2026, 5, 15, 0, 0, 0, 0, loc),
			rule:  DueRuleEndOfNextMonth,
			want:  time.Date(2026, 6, 30, 0, 0, 0, 0, loc),
		},
		{
			name:  "day of current month",
			issue: time.Date(2026, 5, 1, 0, 0, 0, 0, loc),
			rule:  DueRuleDayOfMonth,
			day:   intPtr(15),
			want:  time.Date(2026, 5, 15, 0, 0, 0, 0, loc),
		},
		{
			name:      "day of next month from December",
			issue:     time.Date(2026, 12, 20, 0, 0, 0, 0, loc),
			rule:      DueRuleDayOfMonth,
			day:       intPtr(10),
			nextMonth: true,
			want:      time.Date(2027, 1, 10, 0, 0, 0, 0, loc),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calculateDueDate(c.issue, c.rule, c.day, c.nextMonth)
			if !got.Equal(c.want) {
				t.Errorf("got %s, want %s", got.Format(time.RFC3339), c.want.Format(time.RFC3339))
			}
		})
	}
}

func TestGenerateDueRecurringInvoices_CreatesInvoice(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	template, productID := seedRecurringTemplate(t, testRepo, now.Day(), DueRuleEndOfNextMonth, nil, false)

	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator returned error: %v", err)
	}
	if len(generated) != 1 {
		t.Fatalf("expected 1 invoice generated, got %d", len(generated))
	}

	inv := generated[0]
	if inv.ID == 0 {
		t.Error("generated invoice should have an ID")
	}
	if !inv.IssueDate.Equal(now) {
		t.Errorf("issue date %s != now %s", inv.IssueDate, now)
	}
	wantDue := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if !inv.DueDate.Equal(wantDue) {
		t.Errorf("due date %s != %s", inv.DueDate, wantDue)
	}
	if len(inv.InvoiceLines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(inv.InvoiceLines))
	}
	if inv.InvoiceLines[0].ProductID != productID {
		t.Errorf("line product id mismatch")
	}
	if inv.InvoiceLines[0].Quantity != 2 {
		t.Errorf("line quantity = %d, want 2", inv.InvoiceLines[0].Quantity)
	}

	reloaded, err := testRepo.GetRecurringInvoice(template.ID)
	if err != nil {
		t.Fatalf("reload template: %v", err)
	}
	if reloaded.LastGeneratedAt == nil {
		t.Error("LastGeneratedAt should be set")
	}
}

func setLastGenerated(t *testing.T, repo *Repository, id uint, when time.Time) {
	t.Helper()
	if err := repo.db.Model(&RecurringInvoice{}).Where("id = ?", id).
		Update("last_generated_at", when).Error; err != nil {
		t.Fatalf("set last_generated_at: %v", err)
	}
}

func TestGenerateDueRecurringInvoices_Dedup(t *testing.T) {
	// Month-precision dedup: a second pass on a *different day same month*
	// must still skip.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	first := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	seedRecurringTemplate(t, testRepo, first.Day(), DueRuleEndOfMonth, nil, false)

	if _, err := testRepo.GenerateDueRecurringInvoices(first); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	later := time.Date(2026, 5, 12, 8, 0, 0, 0, time.UTC) // 2 days later, same month
	second, err := testRepo.GenerateDueRecurringInvoices(later)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("month dedup failed: second pass produced %d invoices", len(second))
	}

	all, _ := testRepo.GetInvoices()
	if len(all) != 1 {
		t.Errorf("expected 1 invoice in DB after dedup, got %d", len(all))
	}
}

func TestGenerateDueRecurringInvoices_SkipsInactive(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	template, _ := seedRecurringTemplate(t, testRepo, now.Day(), DueRuleEndOfMonth, nil, false)
	template.IsActive = false
	if err := testRepo.UpdateRecurringInvoice(template); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 0 {
		t.Errorf("expected 0, got %d", len(generated))
	}
}

func TestGenerateDueRecurringInvoices_SkipsWhenScheduledDayNotReached(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	seedRecurringTemplate(t, testRepo, 25, DueRuleEndOfMonth, nil, false) // day 25 still in the future

	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 0 {
		t.Errorf("expected 0 (scheduled day not reached), got %d", len(generated))
	}
}

func TestGenerateDueRecurringInvoices_CatchesUpAfterMissedDay(t *testing.T) {
	// generate_on_day=1 but app didn't run on day 1; today is day 5; LastGeneratedAt is nil.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 5, 8, 0, 0, 0, time.UTC)
	template, _ := seedRecurringTemplate(t, testRepo, 1, DueRuleEndOfMonth, nil, false)

	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 1 {
		t.Fatalf("expected 1 catch-up invoice, got %d", len(generated))
	}

	reloaded, err := testRepo.GetRecurringInvoice(template.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.LastGeneratedAt == nil || reloaded.LastGeneratedAt.Month() != 5 {
		t.Errorf("LastGeneratedAt should be set to May, got %v", reloaded.LastGeneratedAt)
	}
}

func TestGenerateDueRecurringInvoices_RegeneratesNextMonth(t *testing.T) {
	// Already generated last month; new month should regenerate.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	template, _ := seedRecurringTemplate(t, testRepo, 1, DueRuleEndOfMonth, nil, false)
	setLastGenerated(t, testRepo, template.ID, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 1 {
		t.Fatalf("expected 1 invoice in new month, got %d", len(generated))
	}
}

func TestGenerateDueRecurringInvoices_SkipsWhenAlreadyGeneratedThisMonth(t *testing.T) {
	// Already generated earlier this month; subsequent pass must skip.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	template, _ := seedRecurringTemplate(t, testRepo, 1, DueRuleEndOfMonth, nil, false)
	setLastGenerated(t, testRepo, template.ID, time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 0 {
		t.Errorf("expected 0 (already generated this month), got %d", len(generated))
	}
}

func TestGenerateDueRecurringInvoices_EndOfMonthCap_ShortMonth(t *testing.T) {
	// generate_on_day=31 in February (28 days). Should still fire on Feb 28.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 2, 28, 8, 0, 0, 0, time.UTC) // Feb 2026 has 28 days
	seedRecurringTemplate(t, testRepo, 31, DueRuleEndOfMonth, nil, false)

	generated, err := testRepo.GenerateDueRecurringInvoices(now)
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	if len(generated) != 1 {
		t.Fatalf("expected 1 invoice (EOM cap), got %d", len(generated))
	}
	wantDue := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if !generated[0].DueDate.Equal(wantDue) {
		t.Errorf("due date %s != %s", generated[0].DueDate, wantDue)
	}
}

func TestGenerateOne_AtomicClaim(t *testing.T) {
	// Two back-to-back calls with identical inputs must produce exactly one invoice.
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	template, _ := seedRecurringTemplate(t, testRepo, 1, DueRuleEndOfMonth, nil, false)

	inv1, err := testRepo.GenerateRecurringInvoiceNow(template.ID, now)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if inv1 == nil {
		t.Fatal("first call should produce an invoice")
	}

	inv2, err := testRepo.GenerateRecurringInvoiceNow(template.ID, now)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if inv2 != nil {
		t.Error("second call should be rejected by atomic claim")
	}

	all, _ := testRepo.GetInvoices()
	if len(all) != 1 {
		t.Errorf("expected exactly 1 invoice, got %d", len(all))
	}
}

func TestShouldGenerateThisMonth(t *testing.T) {
	loc := time.UTC

	cases := []struct {
		name      string
		genDay    int
		lastGen   *time.Time
		now       time.Time
		want      bool
	}{
		{
			name:   "scheduled day not yet reached",
			genDay: 25,
			now:    time.Date(2026, 5, 10, 0, 0, 0, 0, loc),
			want:   false,
		},
		{
			name:   "scheduled day reached, never generated",
			genDay: 5,
			now:    time.Date(2026, 5, 10, 0, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:    "already generated this month",
			genDay:  1,
			lastGen: timePtr(time.Date(2026, 5, 3, 0, 0, 0, 0, loc)),
			now:     time.Date(2026, 5, 10, 0, 0, 0, 0, loc),
			want:    false,
		},
		{
			name:    "generated previous month, due now",
			genDay:  1,
			lastGen: timePtr(time.Date(2026, 4, 15, 0, 0, 0, 0, loc)),
			now:     time.Date(2026, 5, 10, 0, 0, 0, 0, loc),
			want:    true,
		},
		{
			name:   "EOM cap: gen_day=31 on Feb 28 (last day of Feb)",
			genDay: 31,
			now:    time.Date(2026, 2, 28, 0, 0, 0, 0, loc),
			want:   true,
		},
		{
			name:   "EOM cap: gen_day=31 on Feb 27 (not yet last day)",
			genDay: 31,
			now:    time.Date(2026, 2, 27, 0, 0, 0, 0, loc),
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ri := &RecurringInvoice{
				GenerateOnDay:   c.genDay,
				LastGeneratedAt: c.lastGen,
			}
			got := shouldGenerateThisMonth(ri, c.now)
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestGenerateRecurringInvoiceNow_ManualTrigger(t *testing.T) {
	server, testRepo := setupTestServer(t)
	defer server.Close()

	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	template, _ := seedRecurringTemplate(t, testRepo, 1, DueRuleDayOfMonth, intPtr(15), false)

	inv, err := testRepo.GenerateRecurringInvoiceNow(template.ID, now)
	if err != nil {
		t.Fatalf("manual trigger: %v", err)
	}
	if inv == nil {
		t.Fatal("expected an invoice, got nil")
	}
	wantDue := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	if !inv.DueDate.Equal(wantDue) {
		t.Errorf("due date %s != %s", inv.DueDate, wantDue)
	}

	// Same-day re-trigger: dedup hits.
	again, err := testRepo.GenerateRecurringInvoiceNow(template.ID, now)
	if err != nil {
		t.Fatalf("second manual trigger: %v", err)
	}
	if again != nil {
		t.Error("expected nil on same-day dedup, got an invoice")
	}

	// Different-day-same-month re-trigger: still month-deduped.
	later := time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC)
	stillSameMonth, err := testRepo.GenerateRecurringInvoiceNow(template.ID, later)
	if err != nil {
		t.Fatalf("third manual trigger: %v", err)
	}
	if stillSameMonth != nil {
		t.Error("expected nil on month dedup (different day, same month), got an invoice")
	}
}
