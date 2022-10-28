package main

import (
	"aibizapp/app/dbhandle"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"

	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
)

func CreateFileType(w *multipart.Writer, filename string, fileType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", filename))
	h.Set("Content-Type", fileType)
	return w.CreatePart(h)
}

func UploadDoc(image_data []byte, isAttachment string) {
	if isAttachment == "false" {
		file, err := os.Create("save.png")
		if err != nil {
			fmt.Println("Upload Error: ", err)
			return
		}
		fmt.Println("creating image")
		defer file.Close()
		file.Write(image_data)
	} else {
		file, err := os.Create("/hana/attachment.png")
		if err != nil {
			fmt.Println("Upload Error: ", err)
			return
		}
		defer file.Close()
		file.Write(image_data)
	}
}

func UploadPdfDoc(image_data []byte) string {
	file, err := os.Create("save.pdf")
	if err != nil {
		fmt.Println("Upload Error: ", err)
		return "Upload Error"
	}
	file.Write(image_data)
	doc, err := fitz.New("save.pdf")
	if err != nil {
		panic(err)
	}
	img, err := doc.Image(0)
	if err != nil {
		panic(err)
	}
	f, err := os.Create("save.png")
	if err != nil {
		panic(err)
	}
	err = png.Encode(f, img)
	if err != nil {
		panic(err)
	}
	// convert pdf to png to display on AppGyver
	f.Close()
	file.Close()
	doc.Close()

	return base64_encode("save.png")
}
func base64_encode(imagePath string) string {

	file, _ := os.Open(imagePath)
	fi, _ := file.Stat() //FileInfo interface
	size := fi.Size()    //ファイルサイズ

	data := make([]byte, size)
	file.Read(data)
	file.Close()
	return base64.StdEncoding.EncodeToString(data)
}

func RemoveFile(filepath string) {
	err := os.Remove(filepath)
	if err != nil {
		log.Fatal(err)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "false")
		c.Header("Access-Control-Allow-Headers", "ExtractedId, user_id, document_id, document_name, image_base64, ispdf, doc_id, user_info, isAttchment, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST,HEAD,PATCH, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(CORSMiddleware())
	r.GET("/get_stored_result", handle_id_to_info)
	r.GET("/get_item_result", handle_item_data)
	r.GET("/data/show_data", dbhandle.HandleShowData)
	r.POST("/data/user_login", dbhandle.HandleLogIn)
	r.POST("/create_supplier_invoice", handle_supplier_invoice_creation)
	r.POST("/get_extracted_info", handle_extract_doc_info)
	r.DELETE("/data/delete_data", dbhandle.HandleDeleteData)
	r.DELETE("/delete_data", handle_delete_data)
	s := &http.Server{
		Addr:           ":8000",
		Handler:        r,
		MaxHeaderBytes: 5 << 20,
	}
	s.ListenAndServe()
}
