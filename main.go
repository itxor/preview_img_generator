package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

type gzreadCloser struct {
	*gzip.Reader
	io.Closer
}

func (gz gzreadCloser) Close() error {
	return gz.Closer.Close()
}

// LinkImage ...
type LinkImage struct {
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
}

var (
	filePath string
)

func init() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	filePath = os.Getenv("FILE_PATH")
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	url, err := getUrl()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	fmt.Println("URL: " + url)
	os.Exit(0)
}

func run() error {
	dc := gg.NewContext(1200, 628)

	r, _ := strconv.Atoi(os.Getenv("R"))
	g, _ := strconv.Atoi(os.Getenv("G"))
	b, _ := strconv.Atoi(os.Getenv("B"))
	a, _ := strconv.Atoi(os.Getenv("A"))

	dc.SetColor(color.RGBA{
		uint8(r),
		uint8(g),
		uint8(b),
		uint8(a),
	})
	dc.DrawRectangle(
		0,
		0,
		float64(dc.Width()),
		float64(dc.Height()),
	)
	dc.Fill()

	fontPath := os.Getenv("DESC_FONT")
	if err := dc.LoadFontFace(fontPath, 60); err != nil {
		return errors.Wrap(err, "load font")
	}
	dc.SetColor(color.White)
	s := os.Getenv("RIGHT_SUBTITLE")
	if s == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter right subtitle: ")
		s, _ = reader.ReadString('\n')
		s = s[:len(s)-1]
	}

	marginX := 50.0
	marginY := -10.0
	textWidth, textHeigth := dc.MeasureString(s)
	x := float64(dc.Width()) - textWidth - marginX
	y := float64(dc.Height()) - textHeigth - marginY
	dc.DrawString(s, x, y)

	fontPath = os.Getenv("TITLE_FONT")
	if err := dc.LoadFontFace(fontPath, 60); err != nil {
		return errors.Wrap(err, "load font")
	}
	textColor := color.White
	r_, g_, b_, _ := textColor.RGBA()
	mutedColor := color.RGBA{
		R: uint8(r_),
		G: uint8(g_),
		B: uint8(b_),
		A: uint8(120),
	}
	dc.SetColor(mutedColor)
	s = os.Getenv("LEFT_SUBTITLE")
	if s == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter left subtitle: ")
		s, _ = reader.ReadString('\n')
		s = s[:len(s)-1]
	}
	x = 50
	y = float64(dc.Height()) - 30
	dc.DrawString(s, x, y)

	// читаем заголовок из консоли
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter text: ")
	title, _ := reader.ReadString('\n')

	// Рисуем заголовок
	textShadowColor := color.Black
	textColor = color.White
	fontPath = os.Getenv("DESC_FONT")
	if err := dc.LoadFontFace(fontPath, 90); err != nil {
		return errors.Wrap(err, "load font")
	}

	textRightMargin := 60.0
	textTopMargin := 90.0
	x = textRightMargin
	y = textTopMargin
	maxWidth := float64(dc.Width()) - textRightMargin - textRightMargin
	dc.SetColor(textShadowColor)
	dc.DrawStringWrapped(title, x+1, y+1, 0, 0, maxWidth, 1.5, gg.AlignLeft)
	dc.SetColor(textColor)
	dc.DrawStringWrapped(title, x, y, 0, 0, maxWidth, 1.5, gg.AlignLeft)

	// сохраняем
	if err := dc.SavePNG(filePath); err != nil {
		return errors.Wrap(err, "save png")
	}

	return nil
}

// Возвращает URL изображения на imgbb
func getUrl() (string, error) {
	base64File := getBase64Logo()

	api_key := os.Getenv("BB_KEY")
	url := "https://api.imgbb.com/1/upload?key=" + api_key

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("image", base64File)
	err := writer.Close()
	if err != nil {
		panic(err)
	}

	request, _ := http.NewRequest("POST", url, body)
	request.Header.Add("Cache-Control", "no-cache")
	request.Header.Add("Content-Type", writer.FormDataContentType())
	request.Header.Add("Accept", "*/*")
	request.Header.Add("Accept-Encoding", "gzip, deflate, br")
	request.Header.Add("Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body = gzreadCloser{zr, resp.Body}

	var imageData LinkImage
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&imageData)
	if err != nil {
		return "", err
	}

	return imageData.Data.URL, nil

}

// Возвращает base64 изображения
func getBase64Logo() string {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		panic(err)
	}

	imgFile, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer imgFile.Close()

	fInfo, _ := imgFile.Stat()
	var size int64 = fInfo.Size()
	buf := make([]byte, size)

	fReader := bufio.NewReader(imgFile)
	fReader.Read(buf)

	imgBase64Str := base64.StdEncoding.EncodeToString(buf)

	return imgBase64Str
}
