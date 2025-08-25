package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ID    uint                   `gorm:"primaryKey" json:"id"`
	Name  string                 `gorm:"size:255;not null" json:"name"`
	Lines []RemitInformationLine `gorm:"foreignKey:RemitInformationID" json:"lines"`
}

type RemitInformationLine struct {
	ID                 uint             `gorm:"primaryKey" json:"id"`
	Key                string           `gorm:"size:255;not null" json:"key"`
	Value              string           `gorm:"size:255;not null" json:"value"`
	RemitInformationID uint             `gorm:"not null" json:"remit_information_id"`
	RemitInformation   RemitInformation `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type Product struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Name        string  `gorm:"size:255;not null" json:"name"`
	Description *string `gorm:"type:text" json:"description"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
}

type Company struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"size:255;not null" json:"name"`
	Document string `gorm:"size:30;not null" json:"document"`
	Address  string `gorm:"type:text;not null" json:"address"`
}

type Invoice struct {
	ID                    uint             `gorm:"primaryKey" json:"id"`
	UUID                  uuid.UUID        `gorm:"type:text" json:"uuid"`
	Number                *int             `gorm:"default:0" json:"number"`
	AdditionalInformation *string          `gorm:"type:text" json:"additional_information"`
	Discount              float64          `gorm:"type:decimal(10,2);default:0.00" json:"discount"`
	Penalty               float64          `gorm:"type:decimal(10,2);default:0.00" json:"penalty"`
	Paid                  bool             `gorm:"default:false" json:"paid"`
	IssueDate             time.Time        `gorm:"default:CURRENT_TIMESTAMP" json:"issue_date"`
	DueDate               time.Time        `gorm:"not null" json:"due_date"`
	RemitInformationID    uint             `gorm:"not null" json:"remit_information_id"`
	RemitInformation      RemitInformation `gorm:"constraint:OnDelete:CASCADE" json:"remit_information"`
	CompanyID             uint             `gorm:"not null" json:"company_id"`
	Company               Company          `gorm:"constraint:OnDelete:CASCADE" json:"company"`
	ClientID              uint             `gorm:"not null" json:"client_id"`
	Client                Company          `gorm:"constraint:OnDelete:CASCADE" json:"client"`
	InvoiceLines          []InvoiceLine    `gorm:"foreignKey:InvoiceID" json:"invoice_lines"`
}

func (i *Invoice) Identification() string {
	if i.Number != nil && *i.Number != 0 {
		return strconv.Itoa(*i.Number)
	}

	return i.UUID.String()
}

func (invoice *Invoice) BeforeCreate(tx *gorm.DB) error {
	if invoice.UUID == (uuid.UUID{}) {
		invoice.UUID = uuid.New()
	}
	return nil
}

func (i *Invoice) SubTotal() float64 {
	var subTotal float64
	for _, line := range i.InvoiceLines {
		subTotal += line.Total()
	}
	return subTotal
}

func (i *Invoice) Total() float64 {
	return i.SubTotal() - i.Discount + i.Penalty
}

func (i *Invoice) DueMonth() string {
	return monthsInPortuguese[i.DueDate.Month().String()]
}

func (i *Invoice) Repr() string {
	clientName := strings.ReplaceAll(i.Client.Name, " ", "")
	issueDate := i.IssueDate.Format("20060102")
	return fmt.Sprintf("%s_invoice_%s", clientName, issueDate)
}


type InvoiceLine struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	InvoiceID   uint    `gorm:"not null" json:"invoice_id"`
	Invoice     Invoice `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ProductID   uint    `gorm:"not null" json:"product_id"`
	Product     Product `gorm:"constraint:OnDelete:RESTRICT" json:"product"`
	Quantity    int     `gorm:"default:1;not null" json:"quantity"`
	Description *string `gorm:"size:255" json:"description"`
}

func (il *InvoiceLine) Total() float64 {
	return il.Product.Price * float64(il.Quantity)
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
	return r.db.Select(clause.Associations).Delete(&Company{}, id).Error
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
	// First delete associated lines
	if err := r.db.Where("remit_information_id = ?", id).Delete(&RemitInformationLine{}).Error; err != nil {
		return err
	}
	// Then delete the main record
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
	return r.db.Select(clause.Associations).Delete(&Product{}, id).Error
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First, delete existing invoice lines
		if err := tx.Where("invoice_id = ?", invoice.ID).Delete(&InvoiceLine{}).Error; err != nil {
			return err
		}
		
		// Then save the invoice with new lines
		if err := tx.Save(invoice).Error; err != nil {
			return err
		}
		
		return nil
	})
}

func (r *Repository) GetInvoices() ([]Invoice, error) {
	var invoices []Invoice
	err := r.db.Preload("InvoiceLines.Product").Preload("RemitInformation.Lines").Preload("Company").Preload("Client").Find(&invoices).Error
	return invoices, err
}

func (r *Repository) DeleteInvoice(id uint) error {
	// First delete associated invoice lines
	if err := r.db.Where("invoice_id = ?", id).Delete(&InvoiceLine{}).Error; err != nil {
		return err
	}
	// Then delete the main record
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
