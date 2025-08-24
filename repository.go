package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DATABASE_FILE = "tinycrm.db"

var monthsInPortuguese = map[string]string{
	"January":   "Janeiro",
	"February":  "Fevereiro",
	"March":     "Março",
	"April":     "Abril",
	"May":       "Maio",
	"June":      "Junho",
	"July":      "Julho",
	"August":    "Agosto",
	"September": "Setembro",
	"October":   "Outubro",
	"November":  "Novembro",
	"December":  "Dezembro",
}

type RemitInformation struct {
	ID    uint                   `gorm:"primaryKey"`
	Name  string                 `gorm:"size:255;not null"`
	Lines []RemitInformationLine `gorm:"foreignKey:RemitInformationID"`
}

type RemitInformationLine struct {
	ID                 uint             `gorm:"primaryKey"`
	Key                string           `gorm:"size:255;not null"`
	Value              string           `gorm:"size:255;not null"`
	RemitInformationID uint             `gorm:"not null"`
	RemitInformation   RemitInformation `gorm:"constraint:OnDelete:CASCADE"`
}

type Product struct {
	ID          uint    `gorm:"primaryKey"`
	Name        string  `gorm:"size:255;not null"`
	Description *string `gorm:"type:text"`
	Price       float64 `gorm:"type:decimal(10,2);not null"`
}

type Company struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:255;not null"`
	Document string `gorm:"size:30;not null"`
	Address  string `gorm:"type:text;not null"`
}

type Invoice struct {
	ID                    uint             `gorm:"primaryKey"`
	UUID                  uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid()"`
	Number                *int             `gorm:"default:0"`
	AdditionalInformation *string          `gorm:"type:text"`
	Discount              float64          `gorm:"type:decimal(10,2);default:0.00"`
	Penalty               float64          `gorm:"type:decimal(10,2);default:0.00"`
	IssueDate             time.Time        `gorm:"default:CURRENT_TIMESTAMP"`
	DueDate               time.Time        `gorm:"not null"`
	RemitInformationID    uint             `gorm:"not null"`
	RemitInformation      RemitInformation `gorm:"constraint:OnDelete:CASCADE"`
	CompanyID             uint             `gorm:"not null"`
	Company               Company          `gorm:"constraint:OnDelete:CASCADE"`
	ClientID              uint             `gorm:"not null"`
	Client                Company          `gorm:"constraint:OnDelete:CASCADE"`
	InvoiceLines          []InvoiceLine    `gorm:"foreignKey:InvoiceID"`
}

type InvoiceLine struct {
	ID          uint    `gorm:"primaryKey"`
	InvoiceID   uint    `gorm:"not null"`
	Invoice     Invoice `gorm:"constraint:OnDelete:CASCADE"`
	ProductID   uint    `gorm:"not null"`
	Product     Product `gorm:"constraint:OnDelete:RESTRICT"`
	Quantity    int     `gorm:"default:1;not null"`
	Description *string `gorm:"size:255"`
}

type Repository struct {
	db *gorm.DB
}

func NewRepository() (*Repository, error) {
	return NewRepositoryWithDB(nil)
}

func NewRepositoryWithDB(db *gorm.DB) (*Repository, error) {
	if db == nil {
		var err error
		db, err = gorm.Open(sqlite.Open("tinycrm.db"), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	}
	return &Repository{db: db}, nil
}

func (r *Repository) GetCompany(id uint) (*Company, error) {
	var company Company
	err := r.db.First(&company, id).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *Repository) CreateCompany(company *Company) error {
	return r.db.Create(company).Error
}

func (r *Repository) UpdateCompany(company *Company) error {
	return r.db.Save(company).Error
}

func (r *Repository) GetCompanies() ([]Company, error) {
	var companies []Company
	err := r.db.Find(&companies).Error
	return companies, err
}

func (r *Repository) DeleteCompany(id uint) error {
	return r.db.Delete(&Company{}, id).Error
}

// RemitInformation CRUD
func (r *Repository) GetRemitInformation(id uint) (*RemitInformation, error) {
	var remit RemitInformation
	err := r.db.Preload("Lines").First(&remit, id).Error
	if err != nil {
		return nil, err
	}
	return &remit, nil
}

func (r *Repository) CreateRemitInformation(remit *RemitInformation) error {
	return r.db.Create(remit).Error
}

func (r *Repository) UpdateRemitInformation(remit *RemitInformation) error {
	return r.db.Save(remit).Error
}

func (r *Repository) GetRemitInformations() ([]RemitInformation, error) {
	var remits []RemitInformation
	err := r.db.Preload("Lines").Find(&remits).Error
	return remits, err
}

func (r *Repository) DeleteRemitInformation(id uint) error {
	return r.db.Delete(&RemitInformation{}, id).Error
}

// Product CRUD
func (r *Repository) GetProduct(id uint) (*Product, error) {
	var product Product
	err := r.db.First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *Repository) CreateProduct(product *Product) error {
	return r.db.Create(product).Error
}

func (r *Repository) UpdateProduct(product *Product) error {
	return r.db.Save(product).Error
}

func (r *Repository) GetProducts() ([]Product, error) {
	var products []Product
	err := r.db.Find(&products).Error
	return products, err
}

func (r *Repository) DeleteProduct(id uint) error {
	return r.db.Delete(&Product{}, id).Error
}

// Invoice CRUD
func (r *Repository) GetInvoice(id uint) (*Invoice, error) {
	var invoice Invoice
	err := r.db.Preload("InvoiceLines.Product").Preload("RemitInformation.Lines").Preload("Company").Preload("Client").First(&invoice, id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *Repository) CreateInvoice(invoice *Invoice) error {
	return r.db.Create(invoice).Error
}

func (r *Repository) UpdateInvoice(invoice *Invoice) error {
	return r.db.Save(invoice).Error
}

func (r *Repository) GetInvoices() ([]Invoice, error) {
	var invoices []Invoice
	err := r.db.Preload("InvoiceLines.Product").Preload("RemitInformation.Lines").Preload("Company").Preload("Client").Find(&invoices).Error
	return invoices, err
}

func (r *Repository) DeleteInvoice(id uint) error {
	return r.db.Delete(&Invoice{}, id).Error
}

func (r *Repository) Migrate() {
	fmt.Println("Running migrations...")
	db, err := gorm.Open(sqlite.Open(DATABASE_FILE), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(
		&RemitInformation{},
		&RemitInformationLine{},
		&Product{},
		&Company{},
		&Invoice{},
		&InvoiceLine{},
	)
	fmt.Println("Migrations completed.")
}
