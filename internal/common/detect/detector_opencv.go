//go:build opencv

package detect

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"sort"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
	"golang.org/x/image/draw"
)

var ErrNoContours = fmt.Errorf("no contours found")
var ErrNoCardContours = fmt.Errorf("after restrictions, %w", ErrNoContours)

func NewDetector() Detector {
	return boxDetector{}
}

type boxDetector struct {
}

func (d boxDetector) Detect(img io.Reader) (Images, error) {
	buf := new(bytes.Buffer)

	if _, err := buf.ReadFrom(img); err != nil {
		return nil, fmt.Errorf("failed to read image %w", err)
	}

	resized := new(bytes.Buffer)
	resize(buf, resized, 1024)
	buf = nil

	orig, err := gocv.IMDecode(resized.Bytes(), gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %w", err)
	}
	defer orig.Close()

	normalized := normalizeColors(orig)
	defer normalized.Close()

	candidates, err := findCandidates(orig, normalized)
	if err != nil {
		if errors.Is(err, ErrNoContours) {
			return NewImages(), nil
		}

		return nil, err
	}

	log.Debug().Msgf("found %d candidates", len(candidates))

	return candidates, nil
}

func resize(in io.Reader, out io.Writer, height int) error {
	srcImg, err := jpeg.Decode(in)
	if err != nil {
		return err
	}

	bounds := srcImg.Bounds()
	imgHeight := bounds.Dy()
	if height >= imgHeight {
		if err := jpeg.Encode(out, srcImg, nil); err != nil {
			return err
		}

		return nil
	}

	imgWidth := bounds.Dx()
	ratio := float32(imgWidth) / float32(imgHeight)
	width := int(float32(height) * ratio)

	dstImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dstImg, dstImg.Rect, srcImg, srcImg.Bounds(), draw.Over, nil)
	if err := jpeg.Encode(out, dstImg, nil); err != nil {
		return err
	}

	return nil
}

func normalizeColors(orig gocv.Mat) gocv.Mat {
	// convert to lab color space
	labImg := gocv.NewMat()
	defer labImg.Close()
	gocv.CvtColor(orig, &labImg, gocv.ColorBGRToLab)

	// extract the L channel
	channels := gocv.Split(labImg) // now we have the L image in channels[0]

	clahe := gocv.NewCLAHE()
	dst := gocv.NewMat()
	defer dst.Close()
	// apply the CLAHE algorithm to the L channel
	clahe.Apply(channels[0], &dst)

	// merge the the color planes back into an Lab image
	dst.CopyTo(&channels[0])
	gocv.Merge(channels, &labImg)

	// convert back to BGR
	normalized := gocv.NewMat()
	gocv.CvtColor(labImg, &normalized, gocv.ColorLabToBGR)

	return normalized
}

func findCandidates(orig gocv.Mat, normalized gocv.Mat) (Images, error) {
	contours, err := findContours(orig)
	if err != nil {
		return nil, err
	}
	defer contours.Close()

	images := NewImages()
	for i := 0; i < contours.Size(); i++ {
		pv := contours.At(i)

		img, err := singleCandidate(orig, pv, i)
		if err != nil {
			return nil, err
		}

		images = append(images, Image{img})
	}

	return images, nil
}

func singleCandidate(orig gocv.Mat, pv gocv.PointVector, i int) (image.Image, error) {
	origImg := gocv.NewPointVector()
	defer origImg.Close()
	minR := gocv.MinAreaRect(pv)
	for _, p := range minR.Points {
		origImg.Append(p)
	}

	width := int(math.Min(float64(minR.Height), float64(minR.Width)))
	maxWidth := 200
	maxHeight := 300

	dest := gocv.NewPointVector()
	defer dest.Close()
	dest.Append(image.Pt(0, maxHeight))
	dest.Append(image.Pt(0, 0))
	dest.Append(image.Pt(maxWidth, 0))
	dest.Append(image.Pt(maxWidth, maxHeight))
	if width == minR.Height {
		rotatedImage := gocv.NewPointVector()
		defer rotatedImage.Close()
		rotatedImage.Append(minR.Points[3])
		rotatedImage.Append(minR.Points[0])
		rotatedImage.Append(minR.Points[1])
		rotatedImage.Append(minR.Points[2])
		origImg = rotatedImage
	}

	transform := gocv.GetPerspectiveTransform(origImg, dest)
	perspective := gocv.NewMat()
	defer perspective.Close()
	gocv.WarpPerspective(orig, &perspective, transform, image.Point{X: maxWidth, Y: maxHeight})

	// gocv.IMWrite(fmt.Sprintf("/tmp/match-%v.jpg", i), perspective)
	// log.Debug().Msgf("wrote matched img to %s", fmt.Sprintf("/tmp/match-%v.jpg", i))

	pImg, err := perspective.ToImage()
	if err != nil {
		return nil, fmt.Errorf("failed to create image from perspective %w", err)
	}

	return pImg, nil
}

func findContours(orig gocv.Mat) (gocv.PointsVector, error) {
	blur := gocv.NewMat()
	defer blur.Close()
	gocv.GaussianBlur(orig, &blur, image.Point{7, 7}, 0, 0, gocv.BorderDefault)

	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(blur, &gray, gocv.ColorBGRToGray)

	baseThreshold := 100
	canny := gocv.NewMat()
	defer canny.Close()
	gocv.Canny(blur, &canny, float32(baseThreshold), float32(baseThreshold)*2)

	p5 := image.Point{5, 5}
	kernel := gocv.GetStructuringElement(gocv.MorphRect, p5)

	dilate := gocv.NewMat()
	defer dilate.Close()
	gocv.Dilate(canny, &dilate, kernel)

	threshold := gocv.NewMat()
	defer threshold.Close()
	gocv.Erode(dilate, &threshold, kernel)

	hierarchy := gocv.NewMat()
	defer hierarchy.Close()
	contours := gocv.FindContoursWithParams(threshold, &hierarchy, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	if contours.Size() == 0 {
		return gocv.PointsVector{}, ErrNoContours
	}

	log.Debug().Msgf("found %d contours", contours.Size())

	// sort by area size, biggest first
	areas := make(map[float64]int, 0)
	for i := 0; i < contours.Size(); i++ {
		ca := gocv.ContourArea(contours.At(i))
		areas[ca] = i
	}
	keys := make([]float64, 0, len(areas))
	for k := range areas {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(keys)))

	sHier := make([]gocv.Veci, 0)
	sContours := gocv.NewPointsVector()
	defer sContours.Close()
	for _, k := range keys {
		idx := areas[k]
		sContours.Append(contours.At(idx))
		sHier = append(sHier, hierarchy.GetVeciAt(0, idx))
	}

	cardContours := gocv.NewPointsVector()

	for i := 0; i < sContours.Size(); i++ {
		p := sContours.At(i)
		// parents := sHier[i][3]
		// if parents > 2 {
		// 	// has to many parents
		// 	// log.Debug().Msgf("to many parents %d", parents)
		// 	continue
		// }

		area := gocv.ContourArea(p)
		if area < 3000 {
			// area to small
			continue
		}

		peri := gocv.ArcLength(p, true)
		approx := gocv.ApproxPolyDP(p, 0.01*peri, true)
		if approx.Size() != 4 {
			// no four corner
			continue
		}

		cardContours.Append(p)
	}

	if cardContours.Size() == 0 {
		return gocv.PointsVector{}, ErrNoCardContours
	}

	return cardContours, nil
}
