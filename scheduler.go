package main

import (
	"log"
	"time"
)

const schedulerInterval = 6 * time.Hour

func startRecurringInvoiceScheduler(repo *Repository) {
	go func() {
		log.Printf("scheduler: started, will check every %s", schedulerInterval)
		runRecurringInvoicePass(repo)
		ticker := time.NewTicker(schedulerInterval)
		defer ticker.Stop()
		for range ticker.C {
			runRecurringInvoicePass(repo)
		}
	}()
}

func runRecurringInvoicePass(repo *Repository) {
	now := time.Now()
	generated, err := repo.GenerateDueRecurringInvoices(now)
	if err != nil {
		log.Printf("scheduler: generate failed: %v", err)
		return
	}
	if len(generated) == 0 {
		return
	}
	log.Printf("scheduler: generated %d invoice(s)", len(generated))
	for i := range generated {
		inv := &generated[i]
		if err := sendInvoiceNotification(inv); err != nil {
			log.Printf("scheduler: email for invoice %d failed: %v", inv.ID, err)
		}
	}
}
