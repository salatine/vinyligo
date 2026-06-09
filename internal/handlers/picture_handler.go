package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/disintegration/imaging"
)

const (
	imgbbURL    = "https://api.imgbb.com/1/upload"
	maxWidth    = 1200
	jpegQuality = 70
)

type PictureHandler struct {
	apiKey string
}

func NewPictureHandler(apiKey string) *PictureHandler {
	return &PictureHandler{apiKey: apiKey}
}

func (h *PictureHandler) UploadPicture(picturePath string) (string, error) {
	img, err := imaging.Open(picturePath)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	if bounds.Dx() > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("key", h.apiKey)
	w.WriteField("image", b64)
	w.Close()

	resp, err := http.Post(imgbbURL, w.FormDataContentType(), &body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("imgbb error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	return result.Data.URL, nil
}
