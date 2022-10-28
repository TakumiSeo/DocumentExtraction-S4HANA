package main

import (
	"aibizapp/app/dbhandle"
	"aibizapp/app/hana"
	"bytes"
	"strings"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/joho/godotenv"
)

// To devide process considering computation time(user waiting period)
var itemStore []map[string]interface{}
var headerStore map[string]interface{}

func get_clas_token() (token string) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
	clientid := os.Getenv("clientinfo")
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientid))
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://apjdl.authentication.eu10.hana.ondemand.com/oauth/token?grant_type=client_credentials", nil)
	req.Header.Add("Authorization", auth)
	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, _ := ioutil.ReadAll(rsp.Body)
	defer client.CloseIdleConnections()
	var revToken ReceiveToken
	_ = json.Unmarshal(body, &revToken)
	token = "Bearer " + revToken.Access_token
	return token
}

func handle_extract_doc_info(c *gin.Context) {
	var requestBody ExtractDataBody
	c.BindJSON(&requestBody)
	userId := requestBody.UserId
	docName := requestBody.DocumentName
	imageData := requestBody.ImageBase64
	ispdf := requestBody.IsPdf
	data, _ := base64.StdEncoding.DecodeString(imageData)
	if ispdf == "true" {
		imageData = UploadPdfDoc(data)
	} else {
		UploadDoc(data, "false")
	}
	// set a request
	url := "https://aiservices-dox.cfapps.eu10.hana.ondemand.com/document-information-extraction/v1/document/jobs"
	method := "POST"
	// get a token from init file
	auth := get_clas_token()
	doc_path := "save.png"
	file, err := os.Open(doc_path)
	if err != nil {
		log.Fatal(err)
	}
	// set instances
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	// same as data
	byteFile, err := os.ReadFile(doc_path)
	if err != nil {
		panic(err)
	}
	contentType := http.DetectContentType(byteFile)
	autoFile, _ := CreateFileType(writer, filepath.Base(doc_path), contentType)
	io.Copy(autoFile, file)
	// write option field
	_ = writer.WriteField("options", "{\n		\"extraction\": {\n		  \"headerFields\": [\n			\"documentNumber\",\n			\"taxId\",\n			\"taxName\",\n			\"purchaseOrderNumber\",\n			\"shippingAmount\",\n			\"netAmount\",\n			\"grossAmount\",\n			\"currencyCode\",\n			\"receiverContact\",\n			\"documentDate\",\n			\"taxAmount\",\n			\"taxRate\",\n			\"receiverName\",\n			\"receiverAddress\",\n			\"receiverTaxId\",\n			\"deliveryDate\",\n			\"paymentTerms\",\n			\"deliveryNoteNumber\",\n			\"senderBankAccount\",\n			\"senderAddress\",\n			\"senderName\",\n			\"dueDate\",\n			\"discount\",\n			\"barcode\"\n		  ],\n		  \"lineItemFields\": [\n			\"description\",\n			\"netAmount\",\n			\"quantity\",\n			\"unitPrice\",\n			\"materialNumber\",\n			\"unitOfMeasure\"\n		  ]\n		},\n		\"clientId\": \"c_00\",\n		\"documentType\": \"invoice\",\n		\"receivedDate\": \"2020-02-17\",\n		\"enrichment\": {\n		  \"sender\": {\n			\"top\": 5,\n			\"type\": \"businessEntity\",\n			\"subtype\": \"supplier\"\n		  },\n		  \"employee\": {\n			\"type\": \"employee\"\n		  }\n		}\n	}")
	writer.Close()
	// set a request
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Fatal(err)
		return
	}
	req.Header.Add("Authorization", auth)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
	var revEx RecieveExtractId
	_ = json.Unmarshal(body, &revEx)
	resultId := revEx
	//data store
	dbhandle.HandleDBStoreData(userId, docName, resultId.Id, imageData)
	file.Close()
	if ispdf == "true" {
		RemoveFile("save.pdf")
		RemoveFile(doc_path)
	} else {
		RemoveFile(doc_path)
	}
	c.JSON(200, gin.H{"exId": resultId.Id})
}

func handle_id_to_info(c *gin.Context) {
	id := c.Request.Header.Get("user_info")
	auth := get_clas_token()
	url := "https://aiservices-dox.cfapps.eu10.hana.ondemand.com/document-information-extraction/v1/document/jobs/" + id
	method := "GET"
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Add("Authorization", auth)
	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, _ := ioutil.ReadAll(rsp.Body)
	defer client.CloseIdleConnections()

	buf := bytes.NewBuffer(body)
	j, err := simplejson.NewFromReader(buf)
	if err != nil {
		log.Fatal(err)
	}
	/*
		Make map of document header
		for AppGyver and SupplierInvoice
	*/
	hf := j.Get("extraction").Get("headerFields").MustArray()
	documentContent := map[string]interface{}{
		"country":             j.Get("country").MustString(),
		"documentNumber":      "*Please refresh the page",
		"taxId":               "",
		"taxName":             "",
		"purchaseOrderNumber": "",
		"shippingAmount":      "",
		"netAmount":           "",
		"grossAmount":         "",
		"currencyCode":        "",
		"receiverContact":     "",
		"documentDate":        "",
		"taxAmount":           "",
		"taxRate":             "",
		"receiverName":        "",
		"receiverAddress":     "",
		"receiverTaxId":       "",
		"deliveryDate":        "",
		"paymentTerms":        "",
		"deliveryNoteNumber":  "",
		"senderBankAccount":   "",
		"senderAddress":       "",
		"senderName":          "",
		"dueDate":             "",
		"discount":            "",
		"barcode":             "",
		"taxIsCalculated":     "",
	}

	// initialies taxRate
	taxRate := 0.0
	netAmount := 0.0
	grossAmount := 0.0

	for i := 0; i < len(hf); i++ {
		if hf[i].(map[string]interface{})["rawValue"] != nil {

			if hf[i].(map[string]interface{})["name"].(string) == "taxRate" {

				taxRate, err = hf[i].(map[string]interface{})["value"].(json.Number).Float64()
				if err != nil {
					fmt.Println(err)
				}
				documentContent[hf[i].(map[string]interface{})["name"].(string)] = strconv.FormatFloat(taxRate, 'f', -1, 64)

			} else if hf[i].(map[string]interface{})["name"].(string) == "netAmount" {

				netAmount, err = hf[i].(map[string]interface{})["value"].(json.Number).Float64()
				if err != nil {
					fmt.Println(err)
				}
				documentContent[hf[i].(map[string]interface{})["name"].(string)] = strconv.FormatFloat(netAmount, 'f', -1, 64)
			} else if hf[i].(map[string]interface{})["name"].(string) == "grossAmount" {

				grossAmount, err = hf[i].(map[string]interface{})["value"].(json.Number).Float64()
				if err != nil {
					fmt.Println(err)
				}
				documentContent[hf[i].(map[string]interface{})["name"].(string)] = strconv.FormatFloat(grossAmount, 'f', -1, 64)
			} else if hf[i].(map[string]interface{})["name"].(string) == "documentDate" {

				documentDateVal := hf[i].(map[string]interface{})["value"].(string)
				documentContent[hf[i].(map[string]interface{})["name"].(string)] = documentDateVal

			} else if hf[i].(map[string]interface{})["name"].(string) == "dueDate" {

				dueDateVal := hf[i].(map[string]interface{})["value"].(string)
				documentContent[hf[i].(map[string]interface{})["name"].(string)] = dueDateVal

			} else {

				documentContent[hf[i].(map[string]interface{})["name"].(string)] = hf[i].(map[string]interface{})["rawValue"].(string)

			}
		}
	}

	if documentContent["taxAmount"] == "" {
		taxAmount := 0.0
		if (documentContent["netAmount"] != "") && (documentContent["grossAmount"] != "") {
			taxAmount = grossAmount - netAmount
		} else if (documentContent["netAmount"] != "") && (documentContent["taxRate"] != "") {
			taxAmount = (taxRate/100 + 1) * netAmount
		}
		documentContent["taxAmount"] = strconv.FormatFloat(taxAmount, 'f', -1, 64)
	}

	isCalculated := "false"
	isCalculatedAmount := (taxRate/100 + 1) * netAmount
	if (isCalculatedAmount-1 <= grossAmount) && (grossAmount <= isCalculatedAmount+1) {
		isCalculated = "true"
	}
	documentContent["taxIsCalculated"] = isCalculated
	/*
		Make map of document items
		for AppGyver and SupplierInvoice
		And make AppGyver input -> need to reformat as no nested json
	*/
	liList := []map[string]interface{}{}
	li := j.Get("extraction").Get("lineItems").MustArray()
	for i := 0; i < len(li); i++ {
		documentItems := map[string]interface{}{
			"description":    "",
			"netAmount":      "",
			"quantity":       "",
			"unitPrice":      "",
			"materialNumber": "",
			"unitOfMeasure":  "",
		}
		for _, item := range li[i].([]interface{}) {
			if item.(map[string]interface{})["rawValue"] != nil {
				if item.(map[string]interface{})["name"] == "netAmount" {
					netAmountItem, err := item.(map[string]interface{})["value"].(json.Number).Float64()
					if err != nil {
						fmt.Println(err)
					}
					// no rounding
					// if 2 is to be 2, in case of 12.123, it will be rounded as 12.12
					netAmountItemStr := strconv.FormatFloat(netAmountItem, 'f', -1, 64)
					if err != nil {
						fmt.Println(err)
					}
					documentItems[item.(map[string]interface{})["name"].(string)] = netAmountItemStr
				} else if item.(map[string]interface{})["name"] == "unitPrice" {
					unitPriceItem, err := item.(map[string]interface{})["value"].(json.Number).Float64()
					if err != nil {
						fmt.Println(err)
					}
					// no rounding
					// if 2 is to be 2, in case of 12.123, it will be rounded as 12.12
					unitPriceItemStr := strconv.FormatFloat(unitPriceItem, 'f', -1, 64)
					if err != nil {
						fmt.Println(err)
					}
					documentItems[item.(map[string]interface{})["name"].(string)] = unitPriceItemStr
				} else {

					documentItems[item.(map[string]interface{})["name"].(string)] = item.(map[string]interface{})["rawValue"].(string)
				}
			}
		}
		liList = append(liList, documentItems)
	}

	/*
		Fill quantity
	*/
	qCounter := 0
	for _, itemQ := range liList {
		netAmountItemQBool := false
		unitPriceItemQBool := false
		netAmountItemQ := 0.0
		qualityItemQ := 0.0
		// prevent being inf
		unitPriceItemQ := 1.0
		// Fill out quantity value using netAmount and unitPrice
		if itemQ["quantity"] == "" {
			if itemQ["netAmount"] != "" {
				// need to see if netAmount itself was not extracted
				netAmountItemQ, err = strconv.ParseFloat(itemQ["netAmount"].(string), 64)
				if err != nil {
					fmt.Println(err)
				}
				netAmountItemQBool = true
			}
			if itemQ["unitPrice"] != "" {
				// need to consider if netAmount itself was not extracted
				unitPriceItemQ, err = strconv.ParseFloat(itemQ["unitPrice"].(string), 64)
				if err != nil {
					fmt.Println(err)
				}
				unitPriceItemQBool = true
			}
			if netAmountItemQBool && unitPriceItemQBool {
				liList[qCounter]["quantity"] = strconv.Itoa(int(netAmountItemQ / unitPriceItemQ))
			}
		}
		// Fill out quantity value using netAmount and unitPrice
		qualityItemQBool := false
		unitPriceItemQBool = false
		qualityItemQ = 0.0
		unitPriceItemQ = 0.0
		if itemQ["netAmount"] == "" {
			if itemQ["quality"] != "" {
				// need to see if netAmount itself was not extracted
				qualityItemQ, err = strconv.ParseFloat(itemQ["quality"].(string), 64)
				if err != nil {
					fmt.Println(err)
				}
				qualityItemQBool = true
			}
			if itemQ["unitPrice"] != "" {
				// need to consider if netAmount itself was not extracted
				unitPriceItemQ, err = strconv.ParseFloat(itemQ["unitPrice"].(string), 64)
				if err != nil {
					fmt.Println(err)
				}
				unitPriceItemQBool = true
			}

			if qualityItemQBool && unitPriceItemQBool {
				liList[qCounter]["netAmount"] = strconv.FormatFloat(qualityItemQ*unitPriceItemQ, 'f', -1, 64)
			}
		}
		qCounter += 1
	}

	// currency code setting
	if (len(documentContent["country"].(string)) != 0) && (documentContent["currencyCode"] == "") {
		cCode := currencyConverter(documentContent["country"].(string), (documentContent["country"].(string)))
		documentContent["currencyCode"] = cCode
	}
	itemStore = liList
	headerStore = documentContent
	/*
		for tax detail

		documentListContent := map[string]interface{}{
			"documentContent": documentContent,
			"itemList":        liList,
		}
		map[string]map[string]interface{}{}
		documentListContent["documentContent"] = documentContent
		documentListContent["itemList"] = liList
		fmt.Println(documentContent)
	*/
	c.JSON(200, documentContent)
}

func handle_item_data(c *gin.Context) {
	c.JSON(200, &itemStore)
}

func handle_delete_data(c *gin.Context) {
	var requestBody DleleteId
	c.BindJSON(&requestBody)
	deleteId := requestBody.Value
	deleteArray := [...]string{deleteId}
	payload := map[string]interface{}{
		"value": deleteArray,
	}
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(payload)
	reqBodyBytes.Bytes()
	auth := get_clas_token()
	url := "https://aiservices-dox.cfapps.eu10.hana.ondemand.com/document-information-extraction/v1/document/jobs"
	method := "DELETE"
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, reqBodyBytes)
	req.Header.Add("Authorization", auth)
	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, _ := ioutil.ReadAll(rsp.Body)
	defer client.CloseIdleConnections()
	res := string(body)
	c.JSON(200, res)
}

func handle_supplier_invoice_creation(c *gin.Context) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
	aiBizS4 := os.Getenv("s4clientinfo")
	var requestBody S4Invoice
	c.BindJSON(&requestBody)
	imageData := requestBody.ImageBase64
	documentId := requestBody.DocumentId
	data, _ := base64.StdEncoding.DecodeString(imageData)
	UploadDoc(data, "true")
	docHead := &headerStore
	docItem := &itemStore
	suppInvoiceResult := hana.HandleCreateSuppInvoice(*docHead, *docItem, aiBizS4)
	suppInvoiceMessage := suppInvoiceResult["Message"].(string)
	linkedSAPObjectNum := suppInvoiceResult["LinkedSAPObjectNum"].(string)
	linkedSAPObjectKey := suppInvoiceResult["LinkedSAPObjectKey"].(string)
	attachmentMessage := hana.HandleCreateAttachment(linkedSAPObjectKey, aiBizS4)
	fmt.Println("Created Invoice Document Number " + linkedSAPObjectKey)
	message := ""
	if (suppInvoiceMessage == "Success") && (attachmentMessage == "Success") {
		message = "Success"
	} else if suppInvoiceMessage != "Success" {
		message = suppInvoiceMessage
	} else if attachmentMessage != "Success" {
		message = attachmentMessage
	} else {
		message = "Error"
	}
	RemoveFile("/hana/attachment.png")
	result := map[string]interface{}{
		"Message":            message,
		"LinkedSAPObjectNum": linkedSAPObjectNum,
	}
	dbhandle.HandleUpdateData(documentId, linkedSAPObjectNum)
	c.JSON(200, result)
}

func StructToMap(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	elem := reflect.ValueOf(data).Elem()
	size := elem.NumField()

	for i := 0; i < size; i++ {
		field := elem.Type().Field(i).Name
		value := elem.Field(i).Interface()
		result[field] = value
	}

	return result
}

func currencyConverter(countryCode string, country string) (currencyCode string) {
	file, err := os.Open("data/country-code-to-currency-code-mapping.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	df := dataframe.ReadCSV(file)
	df.Filter(dataframe.F{})
	fil := df.Filter(
		dataframe.F{Colidx: 1, Colname: "CountryCode", Comparator: series.Eq, Comparando: countryCode},
	)
	fil2 := df.Filter(
		dataframe.F{Colidx: 1, Colname: "Country", Comparator: series.Eq, Comparando: country},
	)
	res := strings.Split((fil.Col("Code").Str()), ":")
	res2 := strings.Split((fil2.Col("Code").Str()), ":")
	if strings.Contains(res[3], "Values") {
		code := res[4]
		code = code[:len(code)-1]
		currencyCode = code[2:]
		return currencyCode
	} else {
		if strings.Contains(res2[3], "Values") {
			code := res2[4]
			code = code[:len(code)-1]
			currencyCode = code[2:]
			return currencyCode
		} else {
			return ""
		}
	}

}
