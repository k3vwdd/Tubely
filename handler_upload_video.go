package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
    videoIDString := r.PathValue("videoID")
    videoID, err := uuid.Parse(videoIDString)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
    }

    const maxBytes = 1 << 30

    r.Body = http.MaxBytesReader(w, r.Body, maxBytes,)

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

    err = r.ParseMultipartForm(maxBytes)
    if err != nil {
		respondWithError(w, http.StatusBadRequest, "File to large", nil)
		return
    }


    file, header, err := r.FormFile("video")
    if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid file", nil)
		return
    }

    defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

    tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
    if err != nil {
		respondWithError(w, http.StatusBadRequest, "", nil)
		return
    }


    defer os.Remove(tempFile.Name())
    defer tempFile.Close()

    if _, err = io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "", nil)
		return
    }

    aspectRatio, err  := getVideoAspectRatio(tempFile.Name())
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Couldn't get aspect ratio", nil)
        return

    }

    assetPath := getAssetPath(mediaType)

    if aspectRatio == "16:9" {
        assetPath = "landscape/" + assetPath
    }

    if aspectRatio == "9:16" {
        assetPath = "portrait/" + assetPath
    }


    tempFile.Seek(0, io.SeekStart)

    _, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket: &cfg.s3Bucket,
        Key: &assetPath,
        Body: tempFile,
        ContentType: &mediaType,
    })

    if err != nil {
		respondWithError(w, http.StatusBadRequest, "", nil)
		return
    }

    url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, assetPath)
    video.VideoURL = &url

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)

}

