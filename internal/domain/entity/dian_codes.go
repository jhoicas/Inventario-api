package entity

// CreditNoteConcept representa el código de concepto DIAN para Notas Crédito.
// Tabla oficial DIAN:
//  1: Devolución parcial
//  2: Anulación
//  3: Rebaja
//  4: Descuento
//  5: Rescisión
//  6: Otros
type CreditNoteConcept string

const (
	CreditNoteConceptDevolucionParcial CreditNoteConcept = "1"
	CreditNoteConceptAnulacion         CreditNoteConcept = "2"
	CreditNoteConceptRebaja            CreditNoteConcept = "3"
	CreditNoteConceptDescuento         CreditNoteConcept = "4"
	CreditNoteConceptRescision         CreditNoteConcept = "5"
	CreditNoteConceptOtros             CreditNoteConcept = "6"
)

