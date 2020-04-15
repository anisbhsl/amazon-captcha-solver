package main

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/otiai10/gosseract/v2"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	re  = regexp.MustCompile("[0-9]+")
	re2 = regexp.MustCompile("[A-Z]")
	results=[]string{"BTRPRK","FYKLXE","MXRCTR","MBEPFA","GRJBUY","LACNMU","LXUJEP","JEPMRY","BTCAJK","BXYHJX","HXEJFM","UUGEYE"}
)

func main() {
	for i:=0;i<=11;i++{
		id:=strconv.Itoa(i)
		filePath:="amazon_captcha"+id+".jpg"
		text, err := antiAmazonCaptcha(filePath)
		if err != nil || text!=results[i]{
			fmt.Printf("FILE: %v | NO MATCH | ERROR-> %s \n",filePath,err.Error())
			continue
		}
		fmt.Printf("FILE: %v | MATCH: %v == %v \n",filePath,results[i],text)
	}
}

func antiAmazonCaptcha(imagePath string) (string, error) {
	//read image
	imageFile, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer imageFile.Close()

	var img image.Image
	extensionArr := strings.Split(imagePath, ".")
	if extensionArr[1] == "jpg" || extensionArr[1] == "jpeg" {
		img, _ = jpeg.Decode(imageFile)
	} else if extensionArr[1] == "png" {
		img, _ = png.Decode(imageFile)
	} else {
		return "", fmt.Errorf("unknown image format")
	}

	//use image histogram and threshold to separate 6 letter
	x1 := img.Bounds().Min.X
	x2 := img.Bounds().Max.X
	y1 := img.Bounds().Min.Y
	y2 := img.Bounds().Max.Y
	width := x2 - x1
	height := y2 - y1

	columnMean := []int{}
	for i := 0; i < width; i++ {
		total := 0
		for j := 0; j < height; j++ {
			colorVal := img.At(i, j)
			colorValStr := fmt.Sprintf("%v", colorVal)
			cval, _ := strconv.Atoi(re.FindAllString(colorValStr, 1)[0])
			total += cval
		}
		mean := total / height
		columnMean = append(columnMean, mean)
	}

	retryCount := 0
	threshold := 241
	separatorIndex := []int{}
	for len(separatorIndex) != 7 {
		retryCount += 1
		if threshold >= 255 || threshold <= 0 || retryCount >= 15 {
			return "", fmt.Errorf("can not crop correct number of letters")
		} else if len(separatorIndex) > 7 {
			threshold += 1
		} else if len(separatorIndex) < 7 {
			threshold -= 1
		}

		colMeanIndex := []int{}
		for i, colMean := range columnMean {
			if colMean < threshold {
				colMeanIndex = append(colMeanIndex, i)
			}
		}
		l := len(colMeanIndex)
		colMeanMinIndex := colMeanIndex[0]
		colMeanMaxIndex := colMeanIndex[l-1]

		separatorIndex = []int{colMeanMinIndex}
		for i, val := range colMeanIndex[:l-1] {
			if val != colMeanIndex[i+1]-1 {
				separatorIndex = append(separatorIndex, val)
			}
		}
		separatorIndex = append(separatorIndex, colMeanMaxIndex)
	}

	//rotate each character
	var result string

	for i, val := range separatorIndex[:6] {
		rotateAngle := 0
		if i%2 == 0 {
			rotateAngle = 15
		} else {
			rotateAngle = -15
		}

		//crop and rotate image
		croppedImage := imaging.Crop(img, image.Rect(val+1, 0, separatorIndex[i+1], height))
		rotatedImage := imaging.Rotate(croppedImage, float64(rotateAngle), color.RGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		})

		//pass the image to tesseract
		client := gosseract.NewClient()
		defer client.Close()

		buf := new(bytes.Buffer)
		if extensionArr[1] == "jpg" || extensionArr[1] == "jpeg" {
			_ = jpeg.Encode(buf, rotatedImage, nil)
		} else if extensionArr[1] == "png" {
			_ = png.Encode(buf, rotatedImage)
		}

		client.SetImageFromBytes(buf.Bytes())
		char, _ := client.Text()

		if char == "" {
			return "", fmt.Errorf("got empty letter QQ")
		} else if len(char) > 1 {
			charArr := strings.Split(char, "")
			for _, val := range charArr {
				if re2.Match([]byte(val)) {
					result += val
					break
				}

			}
		} else if !re2.Match([]byte(char)) {
			return "", fmt.Errorf("no english letter %v", char)
		} else {
			result += char
		}

	}
	return result, nil
}
