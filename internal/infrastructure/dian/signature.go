// Servicio de firma digital XAdES-EPES para factura electrónica DIAN.
// Inyecta el nodo ds:Signature dentro de ext:ExtensionContent (segundo UBLExtension).

package dian

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/jhoicas/Inventario-api/pkg/dian"
)

// URL de la política de firma DIAN (XAdES-EPES). Sustituir por la URL oficial si la DIAN la publica.
const (
	SignaturePolicyURL = "https://www.dian.gov.co/contratos/facturaelectronica/v1/Política_de_Firma_Factura_Electrónica_v1.0.pdf"
	NamespaceDS        = "http://www.w3.org/2000/09/xmldsig#"
	NamespaceXAdES     = "http://uri.etsi.org/01903/v1.3.2#"
	AlgC14N            = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	AlgRSASHA256       = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	AlgSHA256          = "http://www.w3.org/2000/09/xmldsig#sha256"
	TransformEnveloped = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
)

// DigitalSignatureService implementa Signer e inyecta la firma en el ExtensionContent.
type DigitalSignatureService struct{}

// NewDigitalSignatureService crea el servicio.
func NewDigitalSignatureService() *DigitalSignatureService {
	return &DigitalSignatureService{}
}

// Sign implementa dian.Signer: firma el XML e inyecta ds:Signature en ext:ExtensionContent.
func (s *DigitalSignatureService) Sign(xmlBytes []byte, cert tls.Certificate) ([]byte, error) {
	if len(xmlBytes) == 0 {
		return nil, fmt.Errorf("dian: XML vacío")
	}
	priv, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("dian: el certificado debe incluir llave privada RSA")
	}

	// 1) Digest del documento completo (sin firma). DIAN puede exigir C14N (Inclusive) antes de hashear.
	docDigest := sha256.Sum256(xmlBytes)
	docDigestB64 := base64.StdEncoding.EncodeToString(docDigest[:])

	// 2) Construir SignedInfo (referencia al documento + política DIAN)
	signedInfoXML, err := s.buildSignedInfo(docDigestB64)
	if err != nil {
		return nil, err
	}

	// 3) Firmar SignedInfo con RSA-SHA256 (canonicalizar SignedInfo en producción)
	signedInfoBytes := []byte(signedInfoXML)
	signHash := sha256.Sum256(signedInfoBytes)
	signatureValue, err := rsa.SignPKCS1v15(nil, priv, crypto.SHA256, signHash[:])
	if err != nil {
		return nil, fmt.Errorf("dian: firmar SignedInfo: %w", err)
	}
	signatureValueB64 := base64.StdEncoding.EncodeToString(signatureValue)

	// 4) Certificado en Base64
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("dian: parsear certificado: %w", err)
	}
	certB64 := base64.StdEncoding.EncodeToString(x509Cert.Raw)

	// 5) Nodo ds:Signature completo (XAdES-EPES con política DIAN)
	signatureXML, err := s.buildSignatureXML(signedInfoXML, signatureValueB64, certB64)
	if err != nil {
		return nil, err
	}

	// 6) Inyectar en el XML: segundo ext:UBLExtension -> ext:ExtensionContent -> ds:Signature
	return s.injectSignature(xmlBytes, signatureXML)
}

// buildSignedInfo genera el XML de SignedInfo (referencia al documento y digest).
func (s *DigitalSignatureService) buildSignedInfo(docDigestB64 string) (string, error) {
	// Referencia URI="" al documento completo; transformada enveloped para excluir la firma.
	sb := &strings.Builder{}
	sb.WriteString(`<ds:SignedInfo xmlns:ds="` + NamespaceDS + `">`)
	sb.WriteString(`<ds:CanonicalizationMethod Algorithm="` + AlgC14N + `"/>`)
	sb.WriteString(`<ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/>`)
	sb.WriteString(`<ds:Reference URI="">`)
	sb.WriteString(`<ds:Transforms><ds:Transform Algorithm="` + TransformEnveloped + `"/>`)
	sb.WriteString(`<ds:Transform Algorithm="` + AlgC14N + `"/></ds:Transforms>`)
	sb.WriteString(`<ds:DigestMethod Algorithm="` + AlgSHA256 + `"/>`)
	sb.WriteString(`<ds:DigestValue>` + docDigestB64 + `</ds:DigestValue>`)
	sb.WriteString(`</ds:Reference>`)
	sb.WriteString(`</ds:SignedInfo>`)
	return sb.String(), nil
}

// buildSignatureXML arma el bloque ds:Signature con SignedInfo, SignatureValue, KeyInfo y política (XAdES-EPES).
func (s *DigitalSignatureService) buildSignatureXML(signedInfoXML, signatureValueB64, certB64 string) (string, error) {
	sb := &strings.Builder{}
	sb.WriteString(`<ds:Signature xmlns:ds="` + NamespaceDS + `" xmlns:xades="` + NamespaceXAdES + `">`)
	sb.WriteString(signedInfoXML)
	sb.WriteString(`<ds:SignatureValue>` + signatureValueB64 + `</ds:SignatureValue>`)
	sb.WriteString(`<ds:KeyInfo><ds:X509Data><ds:X509Certificate>` + certB64 + `</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
	// XAdES-EPES: referencia a la política de firma DIAN
	sb.WriteString(`<ds:Object><xades:QualifyingProperties><xades:SignedProperties>`)
	sb.WriteString(`<xades:SignedSignatureProperties><xades:SignaturePolicyIdentifier>`)
	sb.WriteString(`<xades:SignaturePolicyId><xades:SigPolicyId><xades:Identifier>` + SignaturePolicyURL + `</xades:Identifier></xades:SigPolicyId></xades:SignaturePolicyId>`)
	sb.WriteString(`</xades:SignedSignatureProperties></xades:SignedProperties></xades:QualifyingProperties></ds:Object>`)
	sb.WriteString(`</ds:Signature>`)
	return sb.String(), nil
}

// injectSignature parsea el XML, añade un segundo UBLExtension con ExtensionContent y el ds:Signature, y serializa.
func (s *DigitalSignatureService) injectSignature(xmlBytes []byte, signatureXML string) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return nil, fmt.Errorf("dian: parsear XML: %w", err)
	}
	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("dian: documento sin raíz")
	}

	// Buscar ext:UBLExtensions (Tag puede ser "UBLExtensions" o "ext:UBLExtensions" según el parser)
	nsExt := "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
	var ublExt *etree.Element
	for _, child := range root.ChildElements() {
		tag := child.Tag
		if tag == "ext:UBLExtensions" {
			tag = "UBLExtensions"
		}
		if tag == "UBLExtensions" {
			ublExt = child
			if child.Space != "" {
				nsExt = child.Space
			}
			break
		}
	}
	if ublExt == nil {
		return nil, fmt.Errorf("dian: no se encontró ext:UBLExtensions en el XML")
	}

	// Segundo UBLExtension -> ExtensionContent -> ds:Signature (mismo namespace que el primero)
	secondExt := ublExt.CreateElement("UBLExtension")
	secondExt.Space = nsExt
	extContent := secondExt.CreateElement("ExtensionContent")
	extContent.Space = nsExt

	// Parsear el XML de la firma y añadirlo como hijo de ExtensionContent
	sigDoc := etree.NewDocument()
	if err := sigDoc.ReadFromString(signatureXML); err != nil {
		return nil, fmt.Errorf("dian: parsear nodo Signature: %w", err)
	}
	sigRoot := sigDoc.Root()
	if sigRoot != nil {
		extContent.AddChild(sigRoot)
	}

	var out bytes.Buffer
	doc.WriteTo(&out)
	return out.Bytes(), nil
}

// Asegurar que DigitalSignatureService implementa dian.Signer.
var _ dian.Signer = (*DigitalSignatureService)(nil)
