package handlers

import (
	"testing"

	"github.com/stefanoprivitera/hourglass/internal/models"
)

func TestExpenseCategory_Valid(t *testing.T) {
	categories := []models.ExpenseCategory{
		models.CategoryMileage,
		models.CategoryMeal,
		models.CategoryAccommodation,
		models.CategoryParking,
		models.CategoryOther,
	}

	for _, cat := range categories {
		if !cat.IsValid() {
			t.Errorf("expected %s to be valid", cat)
		}
	}
}
