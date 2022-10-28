package main

type ReceiveToken struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Jti          string `json:"jti"`
}

type ReceiveJobId struct {
	DocumentId string
	Status     string
}

type RecieveExtractId struct {
	Status string `json:"status"`
	Id     string `json:"id"`
}

type DleleteId struct {
	Value string `json:"value"`
}

type ExtractDataBody struct {
	UserId       string `json:"user_id"`
	DocumentName string `json:"document_name"`
	ImageBase64  string `json:"image_base64"`
	IsPdf        string `json:"ispdf"`
}

type S4Invoice struct {
	ImageBase64 string `json:"image_base64"`
	DocumentId  string `json:"document_id"`
}
type DocumentContent struct {
	DocumentNumber      string `json:"documentNumber"`
	TaxId               string `json:"taxId"`
	TaxName             string `json:"taxName"`
	PurchaseOrderNumber string `json:"purchaseOrderNumber"`
	ShippingAmount      string `json:"shippingAmount"`
	NetAmount           string `json:"netAmount"`
	GrossAmount         string `json:"grossAmount"`
	CurrencyCode        string `json:"currencyCode"`
	ReceiverContact     string `json:"receiverContact"`
	DocumentDate        string `json:"documentDate"`
	TaxAmount           string `json:"taxAmount"`
	TaxRate             string `json:"taxRate"`
	ReceiverName        string `json:"receiverName"`
	ReceiverAddress     string `json:"receiverAddress"`
	ReceiverTaxId       string `json:"receiverTaxId"`
	DeliveryDate        string `json:"deliveryDate"`
	PaymentTerms        string `json:"paymentTerms"`
	DeliveryNoteNumber  string `json:"deliveryNoteNumber"`
	SenderBankAccount   string `json:"senderBankAccount"`
	SenderAddress       string `json:"senderAddress"`
	SenderName          string `json:"senderName"`
	DueDate             string `json:"dueDate"`
	Discount            string `json:"discount"`
	Barcode             string `json:"barcode"`
}
