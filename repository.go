package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var DATABASE_FILE = func() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "tinycrm.db"
}()

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

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:255;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
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

const (
	DueRuleEndOfMonth     = "END_OF_MONTH"
	DueRuleEndOfNextMonth = "END_OF_NEXT_MONTH"
	DueRuleDayOfMonth     = "DAY_OF_MONTH"
)

type RecurringInvoice struct {
	ID                        uint                   `gorm:"primaryKey" json:"id"`
	UUID                      uuid.UUID              `gorm:"type:text" json:"uuid"`
	Name                      string                 `gorm:"size:255;not null" json:"name"`
	IsActive                  bool                   `gorm:"default:true" json:"is_active"`
	GenerateOnDay             int                    `gorm:"not null" json:"generate_on_day"`
	DueDateRule               string                 `gorm:"size:32;not null" json:"due_date_rule"`
	DueDayOfMonth             *int                   `json:"due_day_of_month"`
	DueDayInNextMonth         bool                   `gorm:"default:false" json:"due_day_in_next_month"`
	CompanyID                 uint                   `gorm:"not null" json:"company_id"`
	Company                   Company                `gorm:"constraint:OnDelete:CASCADE" json:"company"`
	ClientID                  uint                   `gorm:"not null" json:"client_id"`
	Client                    Company                `gorm:"constraint:OnDelete:CASCADE" json:"client"`
	RemitInformationID        uint                   `gorm:"not null" json:"remit_information_id"`
	RemitInformation          RemitInformation       `gorm:"constraint:OnDelete:CASCADE" json:"remit_information"`
	BaseDiscount              float64                `gorm:"type:decimal(10,2);default:0.00" json:"base_discount"`
	BasePenalty               float64                `gorm:"type:decimal(10,2);default:0.00" json:"base_penalty"`
	BaseAdditionalInformation *string                `gorm:"type:text" json:"base_additional_information"`
	LastGeneratedAt           *time.Time             `json:"last_generated_at"`
	Lines                     []RecurringInvoiceLine `gorm:"foreignKey:RecurringInvoiceID" json:"lines"`
}

type RecurringInvoiceLine struct {
	ID                 uint             `gorm:"primaryKey" json:"id"`
	RecurringInvoiceID uint             `gorm:"not null" json:"recurring_invoice_id"`
	RecurringInvoice   RecurringInvoice `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ProductID          uint             `gorm:"not null" json:"product_id"`
	Product            Product          `gorm:"constraint:OnDelete:RESTRICT" json:"product"`
	Quantity           int              `gorm:"default:1;not null" json:"quantity"`
	Description        *string          `gorm:"size:255" json:"description"`
}

func (ri *RecurringInvoice) BeforeCreate(tx *gorm.DB) error {
	if ri.UUID == (uuid.UUID{}) {
		ri.UUID = uuid.New()
	}
	return nil
}

func calculateDueDate(issueDate time.Time, rule string, dayOfMonth *int, nextMonth bool) time.Time {
	y, m, _ := issueDate.Date()
	loc := issueDate.Location()
	switch rule {
	case DueRuleEndOfMonth:
		return time.Date(y, m+1, 0, 0, 0, 0, 0, loc)
	case DueRuleEndOfNextMonth:
		return time.Date(y, m+2, 0, 0, 0, 0, 0, loc)
	case DueRuleDayOfMonth:
		day := 1
		if dayOfMonth != nil {
			day = *dayOfMonth
		}
		offset := time.Month(0)
		if nextMonth {
			offset = 1
		}
		return time.Date(y, m+offset, day, 0, 0, 0, 0, loc)
	}
	return issueDate
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
		db, err = gorm.Open(sqlite.Open(DATABASE_FILE), &gorm.Config{})
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First, delete existing remit lines
		if err := tx.Where("remit_information_id = ?", remit.ID).Delete(&RemitInformationLine{}).Error; err != nil {
			return err
		}
		
		// Then save the remit information with new lines
		if err := tx.Save(remit).Error; err != nil {
			return err
		}
		
		return nil
	})
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
		var existing Invoice
		if err := tx.Select("uuid", "issue_date").First(&existing, invoice.ID).Error; err != nil {
			return err
		}
		invoice.UUID = existing.UUID
		invoice.IssueDate = existing.IssueDate

		if err := tx.Where("invoice_id = ?", invoice.ID).Delete(&InvoiceLine{}).Error; err != nil {
			return err
		}

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

// RecurringInvoice CRUD
func (r *Repository) GetRecurringInvoice(id uint) (*RecurringInvoice, error) {
	var ri RecurringInvoice
	err := r.db.
		Preload("Lines.Product").
		Preload("RemitInformation.Lines").
		Preload("Company").
		Preload("Client").
		First(&ri, id).Error
	if err != nil {
		return nil, err
	}
	return &ri, nil
}

func (r *Repository) GetRecurringInvoices() ([]RecurringInvoice, error) {
	var ris []RecurringInvoice
	err := r.db.
		Preload("Lines.Product").
		Preload("RemitInformation.Lines").
		Preload("Company").
		Preload("Client").
		Find(&ris).Error
	return ris, err
}

func (r *Repository) CreateRecurringInvoice(ri *RecurringInvoice) error {
	return r.db.Create(ri).Error
}

func (r *Repository) UpdateRecurringInvoice(ri *RecurringInvoice) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing RecurringInvoice
		if err := tx.Select("uuid", "last_generated_at").First(&existing, ri.ID).Error; err != nil {
			return err
		}
		ri.UUID = existing.UUID
		ri.LastGeneratedAt = existing.LastGeneratedAt

		if err := tx.Where("recurring_invoice_id = ?", ri.ID).Delete(&RecurringInvoiceLine{}).Error; err != nil {
			return err
		}
		return tx.Save(ri).Error
	})
}

func (r *Repository) DeleteRecurringInvoice(id uint) error {
	if err := r.db.Where("recurring_invoice_id = ?", id).Delete(&RecurringInvoiceLine{}).Error; err != nil {
		return err
	}
	return r.db.Delete(&RecurringInvoice{}, id).Error
}

// generateOne builds + persists an Invoice from a recurring template, then
// reloads it with relations. Skips if dedup hits (returns nil, nil).
func (r *Repository) generateOne(tx *gorm.DB, ri *RecurringInvoice, now time.Time) (*Invoice, error) {
	// Atomic month claim: only one caller per (template, year-month) succeeds.
	res := tx.Exec(
		`UPDATE recurring_invoices
		    SET last_generated_at = ?
		  WHERE id = ?
		    AND (last_generated_at IS NULL
		         OR strftime('%Y-%m', last_generated_at) != strftime('%Y-%m', ?))`,
		now, ri.ID, now)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}

	dueDate := calculateDueDate(now, ri.DueDateRule, ri.DueDayOfMonth, ri.DueDayInNextMonth)

	lines := make([]InvoiceLine, len(ri.Lines))
	for i, l := range ri.Lines {
		lines[i] = InvoiceLine{
			ProductID:   l.ProductID,
			Quantity:    l.Quantity,
			Description: l.Description,
		}
	}

	inv := Invoice{
		AdditionalInformation: ri.BaseAdditionalInformation,
		Discount:              ri.BaseDiscount,
		Penalty:               ri.BasePenalty,
		IssueDate:             now,
		DueDate:               dueDate,
		RemitInformationID:    ri.RemitInformationID,
		CompanyID:             ri.CompanyID,
		ClientID:              ri.ClientID,
		InvoiceLines:          lines,
	}

	if err := tx.Create(&inv).Error; err != nil {
		return nil, err
	}

	var loaded Invoice
	err := tx.
		Preload("InvoiceLines.Product").
		Preload("RemitInformation.Lines").
		Preload("Company").
		Preload("Client").
		First(&loaded, inv.ID).Error
	if err != nil {
		return nil, err
	}
	return &loaded, nil
}

// shouldGenerateThisMonth reports whether the template is due for generation
// in `now`'s calendar month, given its scheduled day and last-generation
// timestamp. The scheduled day is capped at the last day of the month so that
// generate_on_day=31 still fires on Feb 28 / Apr 30 / etc.
func shouldGenerateThisMonth(t *RecurringInvoice, now time.Time) bool {
	if t.LastGeneratedAt != nil {
		ly, lm, _ := t.LastGeneratedAt.Date()
		ny, nm, _ := now.Date()
		if ly == ny && lm == nm {
			return false
		}
	}
	lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	target := t.GenerateOnDay
	if target > lastDay {
		target = lastDay
	}
	return now.Day() >= target
}

// GenerateDueRecurringInvoices creates invoices for every active template whose
// scheduled day-of-month has been reached this month and which has not yet been
// generated this month. Returns (nil, nil) for templates already claimed for the
// current month.
func (r *Repository) GenerateDueRecurringInvoices(now time.Time) ([]Invoice, error) {
	var templates []RecurringInvoice
	err := r.db.
		Preload("Lines").
		Where("is_active = ?", true).
		Find(&templates).Error
	if err != nil {
		return nil, err
	}

	var generated []Invoice
	for i := range templates {
		t := &templates[i]
		if !shouldGenerateThisMonth(t, now) {
			continue
		}
		err := r.db.Transaction(func(tx *gorm.DB) error {
			inv, err := r.generateOne(tx, t, now)
			if err != nil {
				return err
			}
			if inv != nil {
				generated = append(generated, *inv)
			}
			return nil
		})
		if err != nil {
			return generated, err
		}
	}
	return generated, nil
}

// GenerateRecurringInvoiceNow forces generation for a single template (manual
// trigger). Returns (nil, nil) if already generated this month (atomic month
// claim still applies; bypassing the schedule, but not the dedup).
func (r *Repository) GenerateRecurringInvoiceNow(id uint, now time.Time) (*Invoice, error) {
	var t RecurringInvoice
	if err := r.db.Preload("Lines").First(&t, id).Error; err != nil {
		return nil, err
	}
	var generated *Invoice
	err := r.db.Transaction(func(tx *gorm.DB) error {
		inv, err := r.generateOne(tx, &t, now)
		if err != nil {
			return err
		}
		generated = inv
		return nil
	})
	return generated, err
}

func (r *Repository) Migrate() {
	fmt.Println("Running migrations...")
	db, err := gorm.Open(sqlite.Open(DATABASE_FILE), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(
		&User{},
		&RemitInformation{},
		&RemitInformationLine{},
		&Product{},
		&Company{},
		&Invoice{},
		&InvoiceLine{},
		&RecurringInvoice{},
		&RecurringInvoiceLine{},
	)
	fmt.Println("Migrations completed.")
}

// User CRUD
func (r *Repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *Repository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
