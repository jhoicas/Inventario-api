package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSenderEmail(t *testing.T) {
	tests := []struct {
		name         string
		companyEmail string
		expected     string
	}{
		{
			name:         "Valid email with domain",
			companyEmail: "contacto@artemisa.co",
			expected:     "noresponde@artemisa.co",
		},
		{
			name:         "Different domain",
			companyEmail: "info@ejemplo.com",
			expected:     "noresponde@ejemplo.com",
		},
		{
			name:         "Subdomain",
			companyEmail: "contacto@mail.empresa.co",
			expected:     "noresponde@mail.empresa.co",
		},
		{
			name:         "Empty email",
			companyEmail: "",
			expected:     "",
		},
		{
			name:         "Email without @",
			companyEmail: "contactoartemisa.co",
			expected:     "",
		},
		{
			name:         "Email with @ but no domain",
			companyEmail: "contacto@",
			expected:     "",
		},
		{
			name:         "Multiple @ signs (uses last @)",
			companyEmail: "user@subdomain@ejemplo.com",
			expected:     "noresponde@ejemplo.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSenderEmail(tt.companyEmail)
			assert.Equal(t, tt.expected, result)
		})
	}
}
