// Servicio de firma digital XAdES-EPES para factura electrónica DIAN (Anexo 1.9).
// Inyecta <ds:Signature> en el segundo <ext:ExtensionContent> del XML.

package signer

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/tu-usuario/inventory-pro/pkg/dian"
	"github.com/ucarion/c14n"
)

// DigitalSignatureService implementa la firma XAdES-EPES e inyecta el nodo en el XML.
type DigitalSignatureService struct{}

// NewDigitalSignatureService crea el servicio.
func NewDigitalSignatureService() *DigitalSignatureService {
	return &DigitalSignatureService{}
}

// Sign implementa pkg/dian.Signer. Firma el XML e inyecta ds:Signature en el segundo ExtensionContent.
func (s *DigitalSignatureService) Sign(xmlBytes []byte, cert tls.Certificate) ([]byte, error) {
	if len(xmlBytes) == 0 {
		return nil, fmt.Errorf("dian: XML vacío")
	}
	priv, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("dian: el certificado debe incluir llave privada RSA")
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("dian: parsear certificado: %w", err)
	}

	// 1) Digest del documento (C14N). Reference URI="#invoice-id"
	canonicalDoc, err := canonicalizeXML(xmlBytes)
	if err != nil {
		canonicalDoc = xmlBytes
	}
	docDigest := sha256.Sum256(canonicalDoc)
	docDigestB64 := base64.StdEncoding.EncodeToString(docDigest[:])

	// 2) SignedInfo (C14N, Reference #invoice-id, Digest SHA-256)
	signedInfoXML := s.buildSignedInfo(docDigestB64)
	canonicalSignedInfo, err := canonicalizeXML([]byte(signedInfoXML))
	if err != nil {
		canonicalSignedInfo = []byte(signedInfoXML)
	}
	signHash := sha256.Sum256(canonicalSignedInfo)
	signatureValue, err := rsa.SignPKCS1v15(nil, priv, crypto.SHA256, signHash[:])
	if err != nil {
		return nil, fmt.Errorf("dian: firmar SignedInfo: %w", err)
	}
	signatureValueB64 := base64.StdEncoding.EncodeToString(signatureValue)

	// 3) KeyInfo (X509Certificate)
	certB64 := base64.StdEncoding.EncodeToString(x509Cert.Raw)

	// 4) QualifyingProperties: SignedSignatureProperties (SigningTime, SigningCertificate), SignaturePolicyIdentifier
	signingTime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	certDigestB64, issuerName, serialHex := CertDigestAndIssuerSerial(x509Cert)
	signatureXML := s.buildFullSignature(signedInfoXML, signatureValueB64, certB64, signingTime, certDigestB64, issuerName, serialHex)

	// 5) Inyectar en segundo ext:ExtensionContent
	return s.injectSignature(xmlBytes, signatureXML)
}

func canonicalizeXML(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Entity = map[string]string{}
	return c14n.Canonicalize(dec)
}

func (s *DigitalSignatureService) buildSignedInfo(docDigestB64 string) string {
	uri := "#" + InvoiceElementID
	var sb strings.Builder
	sb.WriteString(`<ds:SignedInfo xmlns:ds="` + NamespaceDS + `">`)
	sb.WriteString(`<ds:CanonicalizationMethod Algorithm="` + AlgC14N + `"/>`)
	sb.WriteString(`<ds:SignatureMethod Algorithm="` + AlgRSASHA256 + `"/>`)
	sb.WriteString(`<ds:Reference URI="` + uri + `">`)
	sb.WriteString(`<ds:Transforms><ds:Transform Algorithm="` + TransformEnveloped + `"/>`)
	sb.WriteString(`<ds:Transform Algorithm="` + AlgC14N + `"/></ds:Transforms>`)
	sb.WriteString(`<ds:DigestMethod Algorithm="` + AlgSHA256 + `"/>`)
	sb.WriteString(`<ds:DigestValue>` + docDigestB64 + `</ds:DigestValue>`)
	sb.WriteString(`</ds:Reference>`)
	sb.WriteString(`</ds:SignedInfo>`)
	return sb.String()
}

func (s *DigitalSignatureService) buildFullSignature(signedInfoXML, signatureValueB64, certB64, signingTime, certDigestB64, issuerName, serialHex string) string {
	var sb strings.Builder
	sb.WriteString(`<ds:Signature xmlns:ds="` + NamespaceDS + `" xmlns:xades="` + NamespaceXAdES + `">`)
	sb.WriteString(signedInfoXML)
	sb.WriteString(`<ds:SignatureValue>` + signatureValueB64 + `</ds:SignatureValue>`)
	sb.WriteString(`<ds:KeyInfo><ds:X509Data><ds:X509Certificate>` + certB64 + `</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
	sb.WriteString(`<ds:Object><xades:QualifyingProperties>`)
	// SignedSignatureProperties: SigningTime, SigningCertificate
	sb.WriteString(`<xades:SignedProperties Id="signed-props">`)
	sb.WriteString(`<xades:SignedSignatureProperties>`)
	sb.WriteString(`<xades:SigningTime>` + signingTime + `</xades:SigningTime>`)
	sb.WriteString(`<xades:SigningCertificate><xades:Cert><xades:CertDigest><ds:DigestMethod Algorithm="` + AlgSHA256 + `"/>`)
	sb.WriteString(`<ds:DigestValue>` + certDigestB64 + `</ds:DigestValue></xades:CertDigest>`)
	sb.WriteString(`<xades:IssuerSerial><ds:X509IssuerName>` + escapeXML(issuerName) + `</ds:X509IssuerName><ds:X509SerialNumber>` + serialHex + `</ds:X509SerialNumber></xades:IssuerSerial></xades:Cert></xades:SigningCertificate>`)
	// SignaturePolicyIdentifier (DIAN v2)
	sb.WriteString(`<xades:SignaturePolicyIdentifier><xades:SignaturePolicyId><xades:SigPolicyId><xades:Identifier>` + SignaturePolicyURLV2 + `</xades:Identifier></xades:SigPolicyId>`)
	if SigPolicyHashDigest != "" {
		sb.WriteString(`<xades:SigPolicyHash><ds:DigestMethod Algorithm="` + AlgSHA256 + `"/><ds:DigestValue>` + SigPolicyHashDigest + `</ds:DigestValue></xades:SigPolicyHash>`)
	}
	sb.WriteString(`</xades:SignaturePolicyId></xades:SignaturePolicyIdentifier>`)
	sb.WriteString(`</xades:SignedSignatureProperties></xades:SignedProperties></xades:QualifyingProperties></ds:Object>`)
	sb.WriteString(`</ds:Signature>`)
	return sb.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func (s *DigitalSignatureService) injectSignature(xmlBytes []byte, signatureXML string) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return nil, fmt.Errorf("dian: parsear XML: %w", err)
	}
	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("dian: documento sin raíz")
	}
	var ublExt *etree.Element
	for _, child := range root.ChildElements() {
		tag := child.Tag
		if tag == "ext:UBLExtensions" {
			tag = "UBLExtensions"
		}
		if tag == "UBLExtensions" {
			ublExt = child
			break
		}
	}
	if ublExt == nil {
		return nil, fmt.Errorf("dian: no se encontró ext:UBLExtensions")
	}
	// Buscar el segundo ext:ExtensionContent (el builder deja el 2.º vacío para la firma)
	var secondExtContent *etree.Element
	var count int
	for _, ext := range ublExt.ChildElements() {
		localTag := ext.Tag
		if localTag == "ext:UBLExtension" {
			localTag = "UBLExtension"
		}
		if localTag != "UBLExtension" {
			continue
		}
		for _, ec := range ext.ChildElements() {
			ecTag := ec.Tag
			if ecTag == "ext:ExtensionContent" {
				ecTag = "ExtensionContent"
			}
			if ecTag == "ExtensionContent" {
				count++
				if count == 2 {
					secondExtContent = ec
					break
				}
			}
		}
		if secondExtContent != nil {
			break
		}
	}
	if secondExtContent == nil {
		return nil, fmt.Errorf("dian: no se encontró el segundo ext:ExtensionContent para inyectar la firma")
	}
	sigDoc := etree.NewDocument()
	if err := sigDoc.ReadFromString(signatureXML); err != nil {
		return nil, fmt.Errorf("dian: parsear Signature: %w", err)
	}
	if sigRoot := sigDoc.Root(); sigRoot != nil {
		secondExtContent.AddChild(sigRoot)
	}
	var out bytes.Buffer
	doc.WriteTo(&out)
	return out.Bytes(), nil
}

var _ dian.Signer = (*DigitalSignatureService)(nil)
