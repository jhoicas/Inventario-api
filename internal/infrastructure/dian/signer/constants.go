// Constantes para firma XAdES-EPES (Anexo Técnico 1.9 DIAN).

package signer

// Política de firma DIAN v2 (obligatoria para XAdES-EPES).
const (
	SignaturePolicyURLV2 = "https://facturaelectronica.dian.gov.co/politicadefirma/v2/politicadefirmav2.pdf"
)

// SigPolicyHashDigest es el SHA-256 del PDF de la política de firma v2 (Base64).
// Hash estándar del documento politicadefirmav2.pdf.
var SigPolicyHashDigest = "dMoMvtcG5aIzgYo0tIsSQeVJBDnUnfSOfBpxXrmor0Y="

// Namespaces y algoritmos XMLDSig / XAdES.
const (
	NamespaceDS    = "http://www.w3.org/2000/09/xmldsig#"
	NamespaceXAdES = "http://uri.etsi.org/01903/v1.3.2#"
	AlgC14N        = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	AlgRSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	AlgSHA256      = "http://www.w3.org/2000/09/xmldsig#sha256"
	TransformEnveloped = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
)

// ID del elemento raíz al que apunta la Reference (debe coincidir con el Id del <Invoice>).
const InvoiceElementID = "invoice-id"
