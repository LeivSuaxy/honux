package validators

import (
	"fmt"
	"strings"
)

func ValidateEmail(email *string) (bool, []error) {
	var validationErrors []error

	if email == nil {
		validationErrors = append(validationErrors, fmt.Errorf("email is nil"))
		return false, validationErrors
	}

	raw := strings.TrimSpace(*email)

	if raw == "" {
		validationErrors = append(validationErrors, fmt.Errorf("email cannot be empty"))
		return false, validationErrors
	}

	if len(raw) < 6 {
		validationErrors = append(validationErrors, fmt.Errorf("email is too short"))
	}

	if len(raw) > 254 {
		validationErrors = append(validationErrors, fmt.Errorf("email is too long"))
	}

	atCount := 0
	atIndex := -1

	for i := 0; i < len(raw); i++ {
		if raw[i] == '@' {
			atCount++
			atIndex = i
		}
	}

	if atCount == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("email must contain exactly one @"))
		return false, validationErrors
	}

	if atCount > 1 {
		validationErrors = append(validationErrors, fmt.Errorf("email must contain only one @"))
		return false, validationErrors
	}

	local := raw[:atIndex]
	domain := raw[atIndex+1:]

	if len(local) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("local part cannot be empty"))
	}

	if len(domain) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("domain cannot be empty"))
	}

	if len(local) > 64 {
		validationErrors = append(validationErrors, fmt.Errorf("local part cannot be longer than 64 characters"))
	}

	if len(domain) > 253 {
		validationErrors = append(validationErrors, fmt.Errorf("domain cannot be longer than 253 characters"))
	}

	if len(local) > 0 {
		if local[0] == '.' {
			validationErrors = append(validationErrors, fmt.Errorf("local part cannot start with a dot"))
		}
		if local[len(local)-1] == '.' {
			validationErrors = append(validationErrors, fmt.Errorf("local part cannot end with a dot"))
		}
		if strings.Contains(local, "..") {
			validationErrors = append(validationErrors, fmt.Errorf("local part cannot contain consecutive dots"))
		}

		for i := 0; i < len(local); i++ {
			c := local[i]
			if (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '.' || c == '_' || c == '%' ||
				c == '+' || c == '-' {
				continue
			}
			validationErrors = append(validationErrors, fmt.Errorf("local part contains invalid character: %q", c))
			break
		}
	}

	if len(domain) > 0 {
		if domain[0] == '.' {
			validationErrors = append(validationErrors, fmt.Errorf("domain cannot start with a dot"))
		}
		if domain[len(domain)-1] == '.' {
			validationErrors = append(validationErrors, fmt.Errorf("domain cannot end with a dot"))
		}
		if strings.Contains(domain, "..") {
			validationErrors = append(validationErrors, fmt.Errorf("domain cannot contain consecutive dots"))
		}

		hasDot := false
		labelLen := 0

		for i := 0; i < len(domain); i++ {
			c := domain[i]

			if c == '.' {
				hasDot = true
				if labelLen == 0 {
					validationErrors = append(validationErrors, fmt.Errorf("domain has an empty label"))
					break
				}
				labelLen = 0
				continue
			}

			if labelLen == 0 && c == '-' {
				validationErrors = append(validationErrors, fmt.Errorf("domain labels cannot start with a hyphen"))
				break
			}

			if !((c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-') {
				validationErrors = append(validationErrors, fmt.Errorf("domain contains invalid character: %q", c))
				break
			}

			labelLen++
			if labelLen > 63 {
				validationErrors = append(validationErrors, fmt.Errorf("a domain label cannot be longer than 63 characters"))
				break
			}
		}

		if labelLen == 0 {
			validationErrors = append(validationErrors, fmt.Errorf("domain cannot end with an empty label"))
		}

		if !hasDot {
			validationErrors = append(validationErrors, fmt.Errorf("domain must contain at least one dot"))
		}
	}

	return len(validationErrors) == 0, validationErrors
}
