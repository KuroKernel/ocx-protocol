package main

)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// validateOffer validates an offer structure
func validateOffer(offer map[string]interface{}) ValidationResult {
	var errors []ValidationError

	// Required fields
	requiredFields := []string{"provider_id", "fleet_id", "unit", "unit_price_amount", "unit_price_currency"}
	for _, field := range requiredFields {
		if _, exists := offer[field]; !exists {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: "required field missing",
			})
		}
	}

	// Validate provider_id
	if providerID, exists := offer["provider_id"]; exists {
		if providerIDStr, ok := providerID.(string); ok {
			if len(providerIDStr) == 0 {
				errors = append(errors, ValidationError{
					Field:   "provider_id",
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "provider_id",
				Message: "must be a string",
			})
		}
	}

	// Validate unit_price_amount
	if priceAmount, exists := offer["unit_price_amount"]; exists {
		if priceStr, ok := priceAmount.(string); ok {
			if price, err := strconv.ParseFloat(priceStr, 64); err != nil || price <= 0 {
				errors = append(errors, ValidationError{
					Field:   "unit_price_amount",
					Message: "must be a positive number",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "unit_price_amount",
				Message: "must be a string",
			})
		}
	}

	// Validate unit_price_currency
	if currency, exists := offer["unit_price_currency"]; exists {
		if currencyStr, ok := currency.(string); ok {
			validCurrencies := []string{"USD", "EUR", "GBP", "JPY"}
			valid := false
			for _, validCurrency := range validCurrencies {
				if currencyStr == validCurrency {
					valid = true
					break
				}
			}
			if !valid {
				errors = append(errors, ValidationError{
					Field:   "unit_price_currency",
					Message: "must be one of: USD, EUR, GBP, JPY",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "unit_price_currency",
				Message: "must be a string",
			})
		}
	}

	// Validate min_hours and max_hours
	if minHours, exists := offer["min_hours"]; exists {
		if minHoursFloat, ok := minHours.(float64); ok {
			if minHoursFloat < 1 || minHoursFloat > 8760 { // Max 1 year
				errors = append(errors, ValidationError{
					Field:   "min_hours",
					Message: "must be between 1 and 8760",
				})
			}
		}
	}

	if maxHours, exists := offer["max_hours"]; exists {
		if maxHoursFloat, ok := maxHours.(float64); ok {
			if maxHoursFloat < 1 || maxHoursFloat > 8760 {
				errors = append(errors, ValidationError{
					Field:   "max_hours",
					Message: "must be between 1 and 8760",
				})
			}
		}
	}

	// Validate min_gpus and max_gpus
	if minGPUs, exists := offer["min_gpus"]; exists {
		if minGPUsFloat, ok := minGPUs.(float64); ok {
			if minGPUsFloat < 1 || minGPUsFloat > 1000 {
				errors = append(errors, ValidationError{
					Field:   "min_gpus",
					Message: "must be between 1 and 1000",
				})
			}
		}
	}

	if maxGPUs, exists := offer["max_gpus"]; exists {
		if maxGPUsFloat, ok := maxGPUs.(float64); ok {
			if maxGPUsFloat < 1 || maxGPUsFloat > 1000 {
				errors = append(errors, ValidationError{
					Field:   "max_gpus",
					Message: "must be between 1 and 1000",
				})
			}
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// validateOrder validates an order structure
func validateOrder(order map[string]interface{}) ValidationResult {
	var errors []ValidationError

	// Required fields
	requiredFields := []string{"buyer_id", "requested_gpus", "hours", "budget_amount", "budget_currency"}
	for _, field := range requiredFields {
		if _, exists := order[field]; !exists {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: "required field missing",
			})
		}
	}

	// Validate buyer_id
	if buyerID, exists := order["buyer_id"]; exists {
		if buyerIDStr, ok := buyerID.(string); ok {
			if len(buyerIDStr) == 0 {
				errors = append(errors, ValidationError{
					Field:   "buyer_id",
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "buyer_id",
				Message: "must be a string",
			})
		}
	}

	// Validate requested_gpus
	if requestedGPUs, exists := order["requested_gpus"]; exists {
		if gpusFloat, ok := requestedGPUs.(float64); ok {
			if gpusFloat < 1 || gpusFloat > 1000 {
				errors = append(errors, ValidationError{
					Field:   "requested_gpus",
					Message: "must be between 1 and 1000",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "requested_gpus",
				Message: "must be a number",
			})
		}
	}

	// Validate hours
	if hours, exists := order["hours"]; exists {
		if hoursFloat, ok := hours.(float64); ok {
			if hoursFloat < 1 || hoursFloat > 8760 {
				errors = append(errors, ValidationError{
					Field:   "hours",
					Message: "must be between 1 and 8760",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "hours",
				Message: "must be a number",
			})
		}
	}

	// Validate budget_amount
	if budgetAmount, exists := order["budget_amount"]; exists {
		if budgetStr, ok := budgetAmount.(string); ok {
			if budget, err := strconv.ParseFloat(budgetStr, 64); err != nil || budget <= 0 {
				errors = append(errors, ValidationError{
					Field:   "budget_amount",
					Message: "must be a positive number",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "budget_amount",
				Message: "must be a string",
			})
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// validateLease validates a lease structure
func validateLease(lease map[string]interface{}) ValidationResult {
	var errors []ValidationError

	// Required fields
	requiredFields := []string{"order_id", "fleet_id", "assigned_gpus"}
	for _, field := range requiredFields {
		if _, exists := lease[field]; !exists {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: "required field missing",
			})
		}
	}

	// Validate assigned_gpus
	if assignedGPUs, exists := lease["assigned_gpus"]; exists {
		if gpusFloat, ok := assignedGPUs.(float64); ok {
			if gpusFloat < 1 || gpusFloat > 1000 {
				errors = append(errors, ValidationError{
					Field:   "assigned_gpus",
					Message: "must be between 1 and 1000",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "assigned_gpus",
				Message: "must be a number",
			})
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// validateParty validates a party structure
func validateParty(party map[string]interface{}) ValidationResult {
	var errors []ValidationError

	// Required fields
	requiredFields := []string{"role", "display_name", "email"}
	for _, field := range requiredFields {
		if _, exists := party[field]; !exists {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: "required field missing",
			})
		}
	}

	// Validate role
	if role, exists := party["role"]; exists {
		if roleStr, ok := role.(string); ok {
			validRoles := []string{"provider", "buyer", "arbiter"}
			valid := false
			for _, validRole := range validRoles {
				if roleStr == validRole {
					valid = true
					break
				}
			}
			if !valid {
				errors = append(errors, ValidationError{
					Field:   "role",
					Message: "must be one of: provider, buyer, arbiter",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "role",
				Message: "must be a string",
			})
		}
	}

	// Validate email
	if email, exists := party["email"]; exists {
		if emailStr, ok := email.(string); ok {
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(emailStr) {
				errors = append(errors, ValidationError{
					Field:   "email",
					Message: "must be a valid email address",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "email",
				Message: "must be a string",
			})
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
