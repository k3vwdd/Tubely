package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Stream struct {
    Width int `json:"width"`
    Height int `json:"height"`
}

type VideoInfo struct {
    Streams []Stream `json:"streams"`
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
    key := make([]byte, 32)
    _, err := rand.Read(key)
    if err != nil {
        log.Fatal("Couldn't create randomn key")
    }
    encodedData := base64.RawURLEncoding.EncodeToString(key)
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", encodedData, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}


func getVideoAspectRatio(filepath string) (string, error) {
    var byt bytes.Buffer
    cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
    cmd.Stdout = &byt
    err := cmd.Run()
    if err != nil {
        return "", fmt.Errorf("ffprobe failed")
    }

    var aspectRatio VideoInfo
    err = json.Unmarshal(byt.Bytes(), &aspectRatio)
    if err != nil {
        return "", fmt.Errorf("unable to transform the data")
    }

    if len(aspectRatio.Streams) == 0 {
        return "", fmt.Errorf("No Streams in result")
    }

    width := aspectRatio.Streams[0].Width
    height := aspectRatio.Streams[0].Height

    if height <= 0 || width <= 0 {
        return "", fmt.Errorf("invalid dimensions")
    }

    a, b := width, height
    for b != 0 {
        a, b = b, a % b
    }

    divider := a
    w := width / divider
    h := height / divider

   ratio := float64(w) / float64(h)

   if ratio >= 1.7 && ratio <= 1.8 {
       return "16:9", nil
   }

   if ratio >= 0.55 && ratio <= 0.57 {
       return "9:16", nil
   }

   return "other", nil
}
