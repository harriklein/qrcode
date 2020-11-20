package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if !exists {
		customHandlerPort = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/qrcodegen", QRCodeGenerator)
	fmt.Println("Go server Listening on: ", customHandlerPort)
	log.Fatal(http.ListenAndServe(":"+customHandlerPort, mux))
}

// setupResponse adds CORS Headers
func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, api_key, Authorization")
}

// QRCodeGenerator generates QRCode
func QRCodeGenerator(w http.ResponseWriter, r *http.Request) {

	setupResponse(&w, r)

	// Variable to store errors
	var err error

	// region Get params ---------------------------------------
	params := r.URL.Query()
	dataString := params.Get("text")
	imageType := strings.ToLower(params.Get("type"))
	imageSize := strings.ToLower(params.Get("size"))
	errorCorrectionLevel := strings.ToUpper(params.Get("ecl"))
	// endregion -----------------------------------------------

	// region Check and split Size into Width and Height. ------
	// Default: without resize it.
	imageSizeA := strings.Split(imageSize, "x")
	imageWidth := 0
	imageHeight := 0
	if len(imageSizeA) > 0 {
		imageWidth, _ = strconv.Atoi(imageSizeA[0])
	}
	if len(imageSizeA) > 1 {
		imageHeight, _ = strconv.Atoi(imageSizeA[1])
	}
	if imageWidth < 0 {
		imageWidth = 0
	}
	if imageHeight < 0 {
		imageHeight = 0
	}
	// endregion -----------------------------------------------

	// region Check ErrorCorrectionLevel -----------------------
	// Default: M
	qrLvl := qr.M
	switch errorCorrectionLevel {
	case "L": // L recovers 7% of data
		qrLvl = qr.L
	case "M": // M recovers 15% of data
		qrLvl = qr.M
	case "Q": // Q recovers 25% of data
		qrLvl = qr.Q
	case "H": // H recovers 30% of data
		qrLvl = qr.H
	}
	// endregion -----------------------------------------------

	// Encode QRCode
	qrCode, _ := qr.Encode(dataString, qrLvl, qr.Auto)

	// Resize QRCode, if requested
	if imageWidth > 0 || imageHeight > 0 {
		if qrCode, err = barcode.Scale(qrCode, imageWidth, imageHeight); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "qr.Scale() failed: %s\n", err)
			return
		}
	}

	switch imageType {
	case "pcl":
		var inBuf, outBuf bytes.Buffer

		if err = png.Encode(&inBuf, qrCode); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "png.Encode() failed: %s\n", err)
			return
		}

		// Use - as input and output to use stdin and stdout.
		cmd := exec.Command("convert", "-", "-monochrome", "pcl:-")
		cmd.Stdin = bytes.NewReader(inBuf.Bytes())
		cmd.Stdout = &outBuf

		if err = cmd.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cmd.Run() failed:  %s\n", err)
			return
		}
		io.Copy(w, bytes.NewReader(outBuf.Bytes()))
	case "jpg", "jpeg":
		jpeg.Encode(w, qrCode, nil)
	default:
		png.Encode(w, qrCode)
	}

}
