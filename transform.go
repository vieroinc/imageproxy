// Copyright 2013 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imageproxy

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"os/exec"

	// register tiff format

	"gopkg.in/h2non/bimg.v1"
	// "gopkg.in/gographics/imagick.v2/imagick"
)

// default compression quality of resized jpegs
const defaultQuality = 95

// maximum distance into image to look for EXIF tags
const maxExifSize = 1 << 20

// resample filter used when resizing images
// var resampleFilter = imaging.Lanczos

// Transform the provided image.  img should contain the raw bytes of an
// encoded image in one of the supported formats (gif, jpeg, or png).  The
// bytes of a similarly encoded image is returned.
func Transform(img []byte, opt Options) ([]byte, error) {
	if !opt.transform() {
		// bail if no transformation was requested
		return img, nil
	}

	// decode image
	m := bimg.NewImage(img)
	format := m.Type()
	if m == nil {
		return nil, errors.New("Could not parse image")
	}

	// apply EXIF orientation for jpeg and tiff source images. Read at most
	// up to maxExifSize looking for EXIF tags.
	if format == "jpeg" || format == "tiff" {

		/*metadata, _ := m.Metadata()
		fmt.Println(metadata.Orientation)
		fmt.Println(metadata.Profile)*/

		exifOpt := exifOrientation(m)
		m.Process(bimg.Options{
			NoAutoRotate: true,
		})
		if exifOpt.transform() {
			err := transformImage(m, exifOpt)
			if err != nil {
				return nil, err
			}
		}
		theImage := m.Image()

		stdout := bytes.NewBuffer([]byte{})
		cmd := exec.Command("exiftool", "-", "-Orientation#=1", "-o", "-")
		cmd.Stdin = bytes.NewReader(theImage)
		cmd.Stdout = stdout

		err := cmd.Run()
		if err != nil {
			fmt.Println(err.Error)
		} else {
			m = bimg.NewImage(stdout.Bytes())
		}

		/*
			image, _ := imaging.Decode(bytes.NewReader(img), imaging.AutoOrientation(true))
			buf := bytes.NewBuffer([]byte{})
			imaging.Encode(buf, image, imaging.JPEG)
			m = bimg.NewImage(buf.Bytes())
		*/
		/*metadata, _ = m.Metadata()
		fmt.Println(metadata.Orientation)
		fmt.Println(metadata.Profile)*/
	}

	// encode webp and tiff as jpeg by default
	if format == "tiff" || format == "webp" {
		format = "jpeg"
	}

	if opt.Format != "" {
		format = opt.Format
	}

	// transform and encode image
	if format != "gif" && format != "jpeg" && format != "png" && format != "tiff" {

		return nil, fmt.Errorf("unsupported format: %v", format)
	}
	var result []byte
	err := transformImage(m, opt)
	if err != nil {
		return nil, err
	}
	switch format {
	case "gif":
		result, err = m.Process(bimg.Options{Type: bimg.GIF})
	case "jpeg":
		quality := opt.Quality
		if quality == 0 {
			quality = defaultQuality
		}
		result, err = m.Process(bimg.Options{Type: bimg.JPEG, Quality: quality, Interlace: true})

		/*
			err = magicwand.ReadImageBlob(result)
			magicwand.SetImageInterlaceScheme(imagick.INTERLACE_JPEG)
			result = magicwand.GetImageBlob();
		*/
	case "png":
		result, err = m.Process(bimg.Options{Type: bimg.PNG})
	case "tiff":
		result, err = m.Process(bimg.Options{Type: bimg.TIFF})
	default:
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// evaluateFloat interprets the option value f. If f is between 0 and 1, it is
// interpreted as a percentage of max, otherwise it is treated as an absolute
// value.  If f is less than 0, 0 is returned.
func evaluateFloat(f float64, max int) int {
	if 0 < f && f < 1 {
		return int(float64(max) * f)
	}
	if f < 0 {
		return 0
	}
	return int(f)
}

// resizeParams determines if the image needs to be resized, and if so, the
// dimensions to resize to.
func performResize(m *bimg.Image, opt Options) error {
	// convert percentage width and height values to absolute values
	size, _ := m.Size()
	imgW := size.Width
	imgH := size.Height
	w := evaluateFloat(opt.Width, imgW)
	h := evaluateFloat(opt.Height, imgH)

	// never resize larger than the original image unless specifically allowed
	if !opt.ScaleUp {
		if w > imgW {
			w = imgW
		}
		if h > imgH {
			h = imgH
		}
	}

	// if requested width and height match the original, skip resizing
	if (w == imgW || w == 0) && (h == imgH || h == 0) {
		return nil
	}

	if opt.Fit {
		_, err := m.Process(bimg.Options{
			Width:  w,
			Height: h,
			Embed:  true,
			Force:  false,
		})
		if err != nil {
			return err
		}
	} else {
		if w == 0 {
			_, err := m.Process(bimg.Options{
				Height: h,
				Crop:   false,
				Force:  false,
			})
			if err != nil {
				return err
			}
		} else if h == 0 {
			_, err := m.Process(bimg.Options{
				Width: w,
				Crop:  false,
				Force: false,
			})
			if err != nil {
				return err
			}
		} else {
			_, err := m.Process(bimg.Options{
				Height: h,
				Width:  w,
				Force:  false,
				Crop:   true,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// var smartcropAnalyzer = smartcrop.NewAnalyzer(nfnt.NewDefaultResizer())

// cropParams calculates crop rectangle parameters to keep it in image bounds
func performCrop(m *bimg.Image, opt Options) error {
	size, _ := m.Size()
	if !opt.SmartCrop && opt.CropX == 0 && opt.CropY == 0 && opt.CropWidth == 0 && opt.CropHeight == 0 {
		return nil
	}

	// width and height of image
	imgW := size.Width
	imgH := size.Height

	if opt.SmartCrop {
		w := evaluateFloat(opt.Width, imgW)
		h := evaluateFloat(opt.Height, imgH)

		log.Printf("smartcrop input: %dx%d", w, h)
		_, err := m.Process(bimg.Options{
			Width:   w,
			Height:  h,
			Force:   false,
			Crop:    true,
			Gravity: bimg.GravityCentre,
		})
		if err != nil {
			log.Printf("error with smartcrop: %v", err)
			return err
		} else {
			r, _ := m.Size()
			log.Printf("smartcrop rectangle: %v", r)
			return nil
		}
	}

	// top left coordinate of crop
	x0 := evaluateFloat(math.Abs(opt.CropX), imgW)
	if opt.CropX < 0 {
		x0 = imgW - x0 // measure from right
	}
	y0 := evaluateFloat(math.Abs(opt.CropY), imgH)
	if opt.CropY < 0 {
		y0 = imgH - y0 // measure from bottom
	}

	// width and height of crop
	w := evaluateFloat(opt.CropWidth, imgW)
	if w == 0 {
		w = imgW
	}
	h := evaluateFloat(opt.CropHeight, imgH)
	if h == 0 {
		h = imgH
	}

	// bottom right coordinate of crop
	x1 := x0 + w
	if x1 > imgW {
		x1 = imgW
	}
	y1 := y0 + h
	if y1 > imgH {
		y1 = imgH
	}

	if x1-x0 != size.Width || y1-y0 != size.Height {
		_, err := m.Process(bimg.Options{
			Left:   x0,
			Top:    y0,
			Width:  x1 - x0,
			Height: y1 - y0,
			Crop:   true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// read EXIF orientation tag from r and adjust opt to orient image correctly.
func exifOrientation(m *bimg.Image) (opt Options) {
	// Exif Orientation Tag values
	// http://sylvana.net/jpegcrop/exif_orientation.html
	const (
		topLeftSide     = 1
		topRightSide    = 2
		bottomRightSide = 3
		bottomLeftSide  = 4
		leftSideTop     = 5
		rightSideTop    = 6
		rightSideBottom = 7
		leftSideBottom  = 8
	)

	metadata, err := m.Metadata()
	if err != nil {
		return opt
	}
	orient := metadata.Orientation

	switch orient {
	case topLeftSide: // 1
		// do nothing
		break
	case topRightSide: // 2
		opt.Rotate = 180
		opt.FlipVertical = true
		break
	case bottomRightSide: // 3
		opt.Rotate = 180
		break
	case bottomLeftSide: // 4
		opt.Rotate = 180
		opt.FlipVertical = true
		break
	case leftSideTop: // 5
		opt.Rotate = 90
		opt.FlipHorizontal = true
		opt.FlipVertical = true
		break
	case rightSideTop: // 6
		opt.Rotate = 90
		break
	case rightSideBottom: // 7
		opt.Rotate = -90
		opt.FlipVertical = true
		opt.FlipHorizontal = true
		break
	case leftSideBottom: // 8
		opt.Rotate = -90
		break
	}
	return opt
}

// transformImage modifies the image m based on the transformations specified
// in opt.
func transformImage(m *bimg.Image, opt Options) error {
	// Parse crop and resize parameters before applying any transforms.
	// This is to ensure that any percentage-based values are based off the
	// size of the original image.

	var err error
	// crop if needed
	performCrop(m, opt)

	// resize if needed
	performResize(m, opt)

	// rotate
	rotate := (int(opt.Rotate) + int(360)) % 360
	switch rotate {
	case 90:
		_, err = m.Rotate(bimg.D90)
		if err != nil {
			return err
		}
	case 180:
		_, err = m.Rotate(bimg.D180)
		if err != nil {
			return err
		}
	case 270:
		_, err = m.Rotate(bimg.D270)
		if err != nil {
			return err
		}
	}

	// flip
	if opt.FlipVertical {
		_, err = m.Flip()
		if err != nil {
			return err
		}
	}
	if opt.FlipHorizontal {
		_, err = m.Flop()
		if err != nil {
			return err
		}
	}

	return nil
}
