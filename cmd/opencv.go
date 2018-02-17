// What it does:
//
// This example opens a video capture device, then streams MJPEG from it.
// Once running point your browser to the hostname/port you passed in the
// command line (for example http://localhost:8080) and you should see
// the live video stream.
//
// How to run:
//
// mjpeg-streamer [camera ID] [host:port]
//
//		go get -u github.com/hybridgroup/mjpeg
// 		go run ./cmd/mjpeg-streamer/main.go 1 0.0.0.0:8080
//

package main

import (
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

var (
	deviceID int
	err      error
	webcam   *gocv.VideoCapture
	stream   *mjpeg.Stream
	vw       *gocv.VideoWriter
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tmjpeg-streamer [camera ID] [host:port]")
		return
	}

	// parse args
	deviceID, _ = strconv.Atoi(os.Args[1])
	host := os.Args[2]

	// open webcam
	webcam, err = gocv.VideoCaptureDevice(deviceID)
	if err != nil {
		fmt.Printf("error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	webcam.Set(gocv.VideoCaptureFOURCC, 1196444237)
	webcam.Set(gocv.VideoCaptureFrameWidth, 1280)
	webcam.Set(gocv.VideoCaptureFrameHeight, 720)
	webcam.Set(gocv.VideoCaptureFPS, 15)

	vw, err = gocv.VideoWriterFile("out.avi", "MJPG", 15, 1280, 720)
	if err != nil {
		panic(err)
	}

	defer vw.Close()

	// create the mjpeg stream
	stream = mjpeg.NewStream()

	// start capturing
	go capture()

	fmt.Println("Capturing. Point your browser to " + host)

	// start http server
	http.Handle("/", stream)
	log.Fatal(http.ListenAndServe(host, nil))
}

type Frame struct {
	Width   int
	Height  int
	MatType gocv.MatType
	Data    []byte
}

func capture() {
	img := gocv.NewMat()
	defer img.Close()

	var firstFrame *gocv.Mat
	var written bool
	var i int

	for {
		i++

		if ok := webcam.Read(img); !ok {
			fmt.Printf("cannot read device %d\n", deviceID)
			return
		}

		if img.Empty() {
			continue
		}

		frame := Frame{
			Width:   img.Cols(),
			Height:  img.Rows(),
			MatType: gocv.MatType(img.Type()),
			Data:    img.ToBytes(),
		}

		gray := gocv.NewMat()
		blur := gocv.NewMat()

		if firstFrame == nil {
			firstFrame = &blur
		}

		gocv.CvtColor(img, gray, gocv.ColorBGRToGray)
		gocv.GaussianBlur(gray, blur, image.Point{X: 25, Y: 25}, 40, 40, gocv.BorderDefault)

		if firstFrame != nil && !written && i == 25 {
			gocv.IMWrite("first.jpg", *firstFrame)
			gocv.IMWrite("next.jpg", blur)

			diff := gocv.NewMat()
			thresh := gocv.NewMat()
			dilated := gocv.NewMat()
			gocv.AbsDiff(*firstFrame, blur, diff)
			gocv.Threshold(diff, thresh, 100, 255, gocv.ThresholdBinary)
			gocv.Dilate(thresh, dilated, gocv.NewMat())
			gocv.Dilate(dilated, dilated, gocv.NewMat())
			gocv.IMWrite("diff.jpg", diff)
			gocv.IMWrite("thresh.jpg", thresh)
			gocv.IMWrite("dilated.jpg", dilated)
			written = true
		}

		i2 := gocv.NewMatFromBytes(frame.Height, frame.Width, frame.MatType, frame.Data)
		vw.Write(i2)
		i2.Close()

		buf, _ := gocv.IMEncode(gocv.JPEGFileExt, img)
		stream.UpdateJPEG(buf)
	}
}
