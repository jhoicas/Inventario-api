package dian

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ── Constantes de entorno ──────────────────────────────────────────────────────

const (
	// AppEnvTest es el identificador de ambiente de habilitación/pruebas DIAN.
	AppEnvTest = "test"
	// AppEnvProd es el identificador de ambiente de producción DIAN.
	AppEnvProd = "prod"
	// AppEnvDev es el identificador local: no envía al WS DIAN.
	AppEnvDev = "dev"

	soapURLTest = "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc"
	soapURLProd = "https://vpfe.dian.gov.co/WcfDianCustomerServices.svc"

	soapNS          = "http://schemas.xmlsoap.org/soap/envelope/"
	soapNSTempuri   = "http://tempuri.org/"
	soapActionBase  = "http://tempuri.org/IWcfDianCustomerServices/"
)

// ── Puerto (interfaz) ──────────────────────────────────────────────────────────

// SubmitResult resultado de la entrega al WS DIAN.
type SubmitResult struct {
	TrackID  string // ZipKey devuelto por SendBillAsync / SendTestSetAsync
	Accepted bool   // true si la DIAN aceptó el documento (HasErrors == false)
	Errors   string // mensajes de error/rechazo de la DIAN (puede ser vacío)
}

// DIANSubmitter define el puerto de salida para la entrega de documentos al WS DIAN.
// La implementación concreta usa SOAP; para tests se puede inyectar un mock.
type DIANSubmitter interface {
	// SubmitZip envía el ZIP del documento electrónico al WS DIAN.
	// env debe ser "test" o "prod"; determina la URL del endpoint.
	// filename es el nombre del archivo ZIP (ej: "900123456SETP000001.zip").
	SubmitZip(ctx context.Context, zipBytes []byte, filename, env string) (*SubmitResult, error)
}

// ── Implementación SOAP ────────────────────────────────────────────────────────

// SOAPDIANClient implementa DIANSubmitter usando el WS SOAP de la DIAN.
// Usa net/http de la stdlib; no requiere librerías de terceros.
type SOAPDIANClient struct {
	httpClient *http.Client
}

// NewSOAPDIANClient construye el cliente SOAP con un timeout de red generoso (60 s)
// ya que el WS DIAN puede tardar varios segundos en responder.
func NewSOAPDIANClient() *SOAPDIANClient {
	return &SOAPDIANClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// ── Estructuras SOAP ──────────────────────────────────────────────────────────

type soapEnvelope struct {
	XMLName xml.Name    `xml:"s:Envelope"`
	XmlnsS  string      `xml:"xmlns:s,attr"`
	XmlnsA  string      `xml:"xmlns:a,attr,omitempty"`
	Header  soapHeader  `xml:"s:Header"`
	Body    soapBody    `xml:"s:Body"`
}

type soapHeader struct{}

type soapBody struct {
	Content interface{}
}

func (b soapBody) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "s:Body"
	e.EncodeToken(start)
	if err := e.Encode(b.Content); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// sendBillAsyncBody cuerpo para la operación SendBillAsync (producción).
type sendBillAsyncBody struct {
	XMLName     xml.Name `xml:"SendBillAsync"`
	Xmlns       string   `xml:"xmlns,attr"`
	FileName    string   `xml:"fileName"`
	ContentFile string   `xml:"contentFile"` // ZIP en Base64
}

// sendTestSetAsyncBody cuerpo para la operación SendTestSetAsync (habilitación).
type sendTestSetAsyncBody struct {
	XMLName     xml.Name `xml:"SendTestSetAsync"`
	Xmlns       string   `xml:"xmlns,attr"`
	FileName    string   `xml:"fileName"`
	ContentFile string   `xml:"contentFile"` // ZIP en Base64
	TestSetID   string   `xml:"testSetId"`   // ID del set de pruebas DIAN (se puede dejar vacío)
}

// ── Estructuras de respuesta SOAP ─────────────────────────────────────────────

type soapResponseEnvelope struct {
	Body soapResponseBody `xml:"Body"`
}

type soapResponseBody struct {
	SendBillResponse    *sendBillAsyncResponse    `xml:"SendBillAsyncResponse"`
	SendTestSetResponse *sendTestSetAsyncResponse `xml:"SendTestSetAsyncResponse"`
	Fault               *soapFault                `xml:"Fault"`
}

type sendBillAsyncResponse struct {
	Result sendBillAsyncResult `xml:"SendBillAsyncResult"`
}

type sendTestSetAsyncResponse struct {
	Result sendBillAsyncResult `xml:"SendTestSetAsyncResult"`
}

type sendBillAsyncResult struct {
	HasErrors        bool     `xml:"HasErrors"`
	ErrorMessageList []string `xml:"ErrorMessageList>string"`
	ZipKey           string   `xml:"ZipKey"`
}

type soapFault struct {
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
}

// ── SubmitZip ─────────────────────────────────────────────────────────────────

// SubmitZip envía el ZIP al WS DIAN usando la operación SOAP correspondiente al entorno.
func (c *SOAPDIANClient) SubmitZip(ctx context.Context, zipBytes []byte, filename, env string) (*SubmitResult, error) {
	soapURL, soapAction, body, err := c.buildRequest(zipBytes, filename, env)
	if err != nil {
		return nil, err
	}

	envelope := soapEnvelope{
		XmlnsS: soapNS,
		Body:   soapBody{Content: body},
	}

	xmlPayload, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("soap: serializar envelope: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, soapURL,
		bytes.NewReader(xmlPayload))
	if err != nil {
		return nil, fmt.Errorf("soap: crear request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("soap: timeout o cancelación: %w", ctx.Err())
		}
		return nil, fmt.Errorf("soap: llamada HTTP fallida: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // max 1 MB
	if err != nil {
		return nil, fmt.Errorf("soap: leer respuesta: %w", err)
	}

	return c.parseResponse(rawBody, env)
}

// buildRequest construye la URL, SOAPAction y body según el entorno.
func (c *SOAPDIANClient) buildRequest(zipBytes []byte, filename, env string) (url, action string, body interface{}, err error) {
	b64Content := base64.StdEncoding.EncodeToString(zipBytes)

	switch env {
	case AppEnvProd:
		url = soapURLProd
		action = soapActionBase + "SendBillAsync"
		body = &sendBillAsyncBody{
			Xmlns:       soapNSTempuri,
			FileName:    filename,
			ContentFile: b64Content,
		}
	case AppEnvTest:
		url = soapURLTest
		action = soapActionBase + "SendTestSetAsync"
		body = &sendTestSetAsyncBody{
			Xmlns:       soapNSTempuri,
			FileName:    filename,
			ContentFile: b64Content,
			TestSetID:   "", // vacío: la DIAN asigna uno automáticamente
		}
	default:
		return "", "", nil, fmt.Errorf("soap: entorno desconocido %q (usar 'test' o 'prod')", env)
	}
	return url, action, body, nil
}

// parseResponse desempaqueta la respuesta SOAP y extrae TrackID y errores.
func (c *SOAPDIANClient) parseResponse(rawBody []byte, env string) (*SubmitResult, error) {
	var envResp soapResponseEnvelope
	if err := xml.Unmarshal(rawBody, &envResp); err != nil {
		// Si no podemos parsear, devolvemos el raw como error pero no abortamos.
		return &SubmitResult{
			Accepted: false,
			Errors:   fmt.Sprintf("no se pudo parsear respuesta SOAP: %s", string(rawBody)),
		}, nil
	}

	// SOAP Fault (error de protocolo, autenticación, etc.)
	if envResp.Body.Fault != nil {
		return &SubmitResult{
			Accepted: false,
			Errors:   fmt.Sprintf("SOAP Fault [%s]: %s", envResp.Body.Fault.FaultCode, envResp.Body.Fault.FaultString),
		}, nil
	}

	var result *sendBillAsyncResult
	if env == AppEnvProd && envResp.Body.SendBillResponse != nil {
		result = &envResp.Body.SendBillResponse.Result
	} else if env == AppEnvTest && envResp.Body.SendTestSetResponse != nil {
		result = &envResp.Body.SendTestSetResponse.Result
	}

	if result == nil {
		return &SubmitResult{
			Accepted: false,
			Errors:   "respuesta SOAP vacía o inesperada: " + string(rawBody),
		}, nil
	}

	errMsg := strings.Join(result.ErrorMessageList, "; ")
	return &SubmitResult{
		TrackID:  result.ZipKey,
		Accepted: !result.HasErrors,
		Errors:   errMsg,
	}, nil
}
