package billing

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrAcquirerNotFound se retorna cuando DIAN responde pero no encuentra el contribuyente.
var ErrAcquirerNotFound = errors.New("contribuyente no encontrado en DIAN")

const (
	dianSoapURLTest = "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc"
	dianSoapURLProd = "https://vpfe.dian.gov.co/WcfDianCustomerServices.svc"
	dianSOAPAction  = `"http://wcf.dian.colombia/IWcfDianCustomerServices/GetAcquirer"`
)

// AcquirerInfo contiene los datos del contribuyente retornados por la DIAN.
// valid idType: 11,12,13,21,22,31,41,42,47,48,50,91
type AcquirerInfo struct {
	BusinessName      string `json:"business_name"`
	TaxID             string `json:"tax_id"`
	IDCode            string `json:"id_code"`
	DV                string `json:"dv"`
	TypeLiability     string `json:"type_liability"`
	TypeOrganization  string `json:"type_organization"`
	TypeRegime        string `json:"type_regime"`
	TaxScheme         string `json:"tax_scheme"`
	StatusCode        string `json:"status_code"`
	StatusDescription string `json:"status_description"`
}

// ── SOAP XML structs ──────────────────────────────────────────────────────────

type acquirerEnvelope struct {
	XMLName xml.Name     `xml:"Envelope"`
	Body    acquirerBody `xml:"Body"`
}

type acquirerBody struct {
	Response acquirerResponse `xml:"GetAcquirerResponse"`
}

type acquirerResponse struct {
	Result acquirerResult `xml:"GetAcquirerResult"`
}

type acquirerResult struct {
	IsValid           string `xml:"IsValid"`
	StatusCode        string `xml:"StatusCode"`
	StatusDescription string `xml:"StatusDescription"`
	BusinessName      string `xml:"businessName"`
	TaxID             string `xml:"taxId"`
	IDCode            string `xml:"idCode"`
	DV                string `xml:"dv"`
	TypeLiability     string `xml:"typeLiability"`
	TypeOrganization  string `xml:"typeOrganization"`
	TypeRegime        string `xml:"typeRegime"`
	TaxScheme         string `xml:"taxScheme"`
}

var dianHTTPClient = &http.Client{Timeout: 30 * time.Second}

// GetAcquirer consulta WcfDianCustomerServices para obtener la información
// de un contribuyente por tipo y número de documento.
//
// url: endpoint SOAP de la DIAN (vpfe-hab o vpfe)
// cert: certificado para autenticación (puede estar vacío si no se requiere)
func GetAcquirer(ctx context.Context, url, idType, idNumber string, cert string) (*AcquirerInfo, error) {
	if url == "" {
		url = dianSoapURLTest
	}

	soapBody := fmt.Sprintf(
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <GetAcquirer xmlns="http://wcf.dian.colombia">
      <accountCode>%s</accountCode>
      <accountCodeT>%s</accountCodeT>
    </GetAcquirer>
  </s:Body>
</s:Envelope>`,
		xmlEscape(idNumber),
		xmlEscape(idType),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(soapBody))
	if err != nil {
		return nil, fmt.Errorf("dian_ws: construir petición: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", dianSOAPAction)

	// Si hay certificado, podría usarse para autenticación cliente (TLS mutual)
	// Por ahora se ignora; puede implementarse en versiones futuras
	_ = cert

	resp, err := dianHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dian_ws: llamada SOAP: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dian_ws: leer respuesta: %w", err)
	}

	var envelope acquirerEnvelope
	if err := xml.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("dian_ws: parsear XML: %w", err)
	}

	r := envelope.Body.Response.Result
	if r.IsValid != "true" {
		return nil, fmt.Errorf("%w: %s", ErrAcquirerNotFound, r.StatusDescription)
	}

	return &AcquirerInfo{
		BusinessName:      r.BusinessName,
		TaxID:             r.TaxID,
		IDCode:            r.IDCode,
		DV:                r.DV,
		TypeLiability:     r.TypeLiability,
		TypeOrganization:  r.TypeOrganization,
		TypeRegime:        r.TypeRegime,
		TaxScheme:         r.TaxScheme,
		StatusCode:        r.StatusCode,
		StatusDescription: r.StatusDescription,
	}, nil
}

// xmlEscape reemplaza los cinco caracteres especiales XML para prevenir inyección.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s)) //nolint:errcheck
	return buf.String()
}
