package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPhase2ModelsIncludeFlattenedEntryFields(t *testing.T) {
	customerID := uuid.New()
	projectID := uuid.New()
	userID := uuid.New()
	rate := 12.34
	now := time.Now().UTC()
	expenseType := ExpenseCategory("parking")

	contract := Contract{CustomerID: &customerID}
	if contract.CustomerID == nil || *contract.CustomerID != customerID {
		t.Fatalf("expected contract customer id to round-trip")
	}

	settings := OrganizationSettings{
		OrganizationID:       uuid.New(),
		DefaultKmRate:        &rate,
		Currency:             "EUR",
		WeekStartDay:         1,
		Timezone:             "UTC",
		ShowApprovalHistory:   true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if settings.Currency != "EUR" || !settings.ShowApprovalHistory {
		t.Fatalf("expected organization settings fields to be populated")
	}

	manager := ProjectManager{ProjectID: projectID, UserID: userID}
	if manager.ProjectID != projectID || manager.UserID != userID {
		t.Fatalf("expected project manager fields to be populated")
	}

	timeEntry := TimeEntry{
		ProjectID:   &projectID,
		Hours:       &rate,
		Description: "design work",
		DeletedAt:   &now,
	}
	if timeEntry.ProjectID == nil || *timeEntry.Hours != rate || timeEntry.Description != "design work" {
		t.Fatalf("expected flattened time entry fields to be populated")
	}

	if !expenseType.IsValid() {
		t.Fatalf("expected new expense type %q to be valid", expenseType)
	}

	expense := Expense{
		ProjectID:   &projectID,
		CustomerID:  &customerID,
		Type:        &expenseType,
		Amount:      &rate,
		KmDistance:  &rate,
		Description: "parking fee",
		DeletedAt:   &now,
	}
	if expense.ProjectID == nil || expense.CustomerID == nil || expense.Type == nil || *expense.Amount != rate {
		t.Fatalf("expected flattened expense fields to be populated")
	}

	receipt := ExpenseReceipt{ReceiptData: []byte("receipt"), MimeType: "image/png"}
	if len(receipt.ReceiptData) == 0 || receipt.MimeType != "image/png" {
		t.Fatalf("expected receipt blob fields to be populated")
	}
}

