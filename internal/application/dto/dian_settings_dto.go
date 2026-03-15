package dto

import "time"

// UpsertDIANSettingsRequest representa los datos para guardar configuración DIAN.
type UpsertDIANSettingsRequest struct {
	Environment         string
	CertificateFileName string
	CertificateData     []byte
	CertificatePassword string
}

// DIANSettingsResponse representa la configuración DIAN guardada para la empresa.
type DIANSettingsResponse struct {
	CompanyID           string    `json:"company_id"`
	Environment         string    `json:"environment"`
	CertificateFileName string    `json:"certificate_file_name"`
	CertificateFileSize int64     `json:"certificate_file_size"`
	UpdatedAt           time.Time `json:"updated_at"`
}
