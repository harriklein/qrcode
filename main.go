package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os/exec"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

func main() {
	http.HandleFunc("/", viewCodeHandler)
	http.ListenAndServe(":8888", nil)
}

func viewCodeHandler(w http.ResponseWriter, r *http.Request) {

	var d struct {
		Message     string `json:"message"`
		ImageType   string `json:"imageType"`
		ImageWidth  int    `json:"imageWidth"`
		ImageHeight int    `json:"imageHeight"`
	}

	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "json.Decoder() failed with %s\n", err)
		return
	}

	dataString := d.Message
	imageType := d.ImageType
	imageWidth := d.ImageWidth
	imageHeight := d.ImageHeight

	if dataString == "" {
		dataString = "QRCode!"
	}
	if imageWidth < 1 {
		imageWidth = 512
	}
	if imageHeight < 1 {
		imageHeight = 512
	}
	if (imageType != "pcl") && (imageType != "jpg") && (imageType != "jpeg") {
		imageType = "png"
	}

	qrCode, _ := qr.Encode(dataString, qr.L, qr.Auto)
	qrCode, _ = barcode.Scale(qrCode, imageWidth, imageHeight)

	if imageType == "png" {
		png.Encode(w, qrCode)
	} else if (imageType == "jpg") || (imageType == "jpeg") {
		jpeg.Encode(w, qrCode, nil)
	} else {
		var b bytes.Buffer
		var bOut bytes.Buffer

		if err := png.Encode(&b, qrCode); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "png.Encode() failed with %s\n", err)
			return
		}

		// Use - as input and output to use stdin and stdout.
		cmd := exec.Command("convert", "-", "-monochrome", "pcl:-")
		cmd.Stdin = bytes.NewReader(b.Bytes())
		cmd.Stdout = &bOut
		err := cmd.Run()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cmd.Run() failed with %s\n", err)
			return
		}
		io.Copy(w, bytes.NewReader(bOut.Bytes()))
	}
}
