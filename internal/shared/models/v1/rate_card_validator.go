package v1

import (
	"context"
	"fmt"
	"time"
)

// RateCardValidator provides validation logic for rate cards
type RateCardValidator struct{}

// NewRateCardValidator creates a new rate card validator
func NewRateCardValidator() *RateCardValidator {
	return &RateCardValidator{}
}

// ValidateNoOverlap checks if a rate card overlaps with existing active rate cards
func (v *RateCardValidator) ValidateNoOverlap(
	ctx context.Context,
	partnerCode string,
	effectiveFrom time.Time,
	effectiveTo *time.Time,
	excludeID *string,
	existingCards []*UnifiedRateCard,
) error {
	// Filter existing cards for the same partner
	relevantCards := make([]*UnifiedRateCard, 0)
	for _, card := range existingCards {
		// Skip if it's the same card being updated
		if excludeID != nil && card.ID.String() == *excludeID {
			continue
		}

		// Only check active cards for the same partner
		if card.PartnerCode == partnerCode && card.IsActive {
			relevantCards = append(relevantCards, card)
		}
	}

	// Check for overlaps
	for _, existing := range relevantCards {
		if v.hasOverlap(effectiveFrom, effectiveTo, existing.EffectiveFrom, existing.EffectiveTo) {
			return fmt.Errorf(
				"rate card has overlapping effective dates with existing rate card '%s' (ID: %s). "+
					"Existing: %s to %s, New: %s to %s",
				existing.Name,
				existing.ID.String(),
				existing.EffectiveFrom.Format("2006-01-02"),
				formatTimePtr(existing.EffectiveTo),
				effectiveFrom.Format("2006-01-02"),
				formatTimePtr(effectiveTo),
			)
		}
	}

	return nil
}

// hasOverlap checks if two date ranges overlap
// Range 1: from1 to to1 (to1 can be nil meaning infinity)
// Range 2: from2 to to2 (to2 can be nil meaning infinity)
func (v *RateCardValidator) hasOverlap(from1 time.Time, to1 *time.Time, from2 time.Time, to2 *time.Time) bool {
	// If both end dates are nil, they definitely overlap
	if to1 == nil && to2 == nil {
		return true
	}

	// If first range has no end date
	if to1 == nil {
		// Overlaps if second range ends after first range starts
		if to2 == nil {
			return true // Both infinite
		}
		return to2.After(from1) || to2.Equal(from1)
	}

	// If second range has no end date
	if to2 == nil {
		// Overlaps if second range starts before first range ends
		return from2.Before(*to1) || from2.Equal(*to1)
	}

	// Both ranges have end dates
	// Overlap if: from1 < to2 AND from2 < to1
	return (from1.Before(*to2) || from1.Equal(*to2)) &&
		(from2.Before(*to1) || from2.Equal(*to1))
}

// ValidateDefaultRateCard ensures only one default rate card exists per partner
func (v *RateCardValidator) ValidateDefaultRateCard(
	ctx context.Context,
	partnerCode string,
	isDefault bool,
	excludeID *string,
	existingCards []*UnifiedRateCard,
) error {
	if !isDefault {
		return nil // Not setting as default, no validation needed
	}

	// Check if another default exists
	for _, card := range existingCards {
		// Skip if it's the same card being updated
		if excludeID != nil && card.ID.String() == *excludeID {
			continue
		}

		if card.PartnerCode == partnerCode && card.IsDefault && card.IsActive {
			return fmt.Errorf(
				"another default rate card already exists for partner '%s': '%s' (ID: %s). "+
					"Please unset the existing default before setting a new one",
				partnerCode,
				card.Name,
				card.ID.String(),
			)
		}
	}

	return nil
}

// ValidateDateRange ensures effective_from is before effective_to
func (v *RateCardValidator) ValidateDateRange(effectiveFrom time.Time, effectiveTo *time.Time) error {
	if effectiveTo == nil {
		return nil // No end date is valid
	}

	if effectiveTo.Before(effectiveFrom) || effectiveTo.Equal(effectiveFrom) {
		return fmt.Errorf(
			"effective_to (%s) must be after effective_from (%s)",
			effectiveTo.Format("2006-01-02"),
			effectiveFrom.Format("2006-01-02"),
		)
	}

	return nil
}

// ValidateEffectiveDates runs all date-related validations
func (v *RateCardValidator) ValidateEffectiveDates(
	ctx context.Context,
	partnerCode string,
	effectiveFrom time.Time,
	effectiveTo *time.Time,
	isDefault bool,
	excludeID *string,
	existingCards []*UnifiedRateCard,
) error {
	// Validate date range
	if err := v.ValidateDateRange(effectiveFrom, effectiveTo); err != nil {
		return err
	}

	// Validate no overlap
	if err := v.ValidateNoOverlap(ctx, partnerCode, effectiveFrom, effectiveTo, excludeID, existingCards); err != nil {
		return err
	}

	// Validate default card
	if err := v.ValidateDefaultRateCard(ctx, partnerCode, isDefault, excludeID, existingCards); err != nil {
		return err
	}

	return nil
}

// formatTimePtr formats a time pointer for display
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "infinity"
	}
	return t.Format("2006-01-02")
}

