package entity

import "time"

// DIANSettings almacena la configuración de certificado DIAN por empresa.
type DIANSettings struct {
	CompanyID                    string
	Environment                  string // test | prod
	CertificatePath              string
	CertificateFileName          string
	CertificateFileSize          int64
	CertificatePasswordEncrypted string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}
