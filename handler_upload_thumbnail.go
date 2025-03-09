package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

    // multiply by powers of 2. 10 << 20 is the same as 10 * 1024 * 1024, which is 10MB
    const maxMemory = 10 << 20
    r.ParseMultipartForm(maxMemory)

    file, header, err := r.FormFile("thumbnail")
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
        return
    }

    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Unable to read data from thumbnail", err)
        return
    }

    mediaType := header.Header.Get("Content-Type")
    if mediaType == "" {
        mediaType = "image/png"
    }

    dataEncoded := base64.StdEncoding.EncodeToString(data)
    // The format is:data:<media-type>;base64,<data>
    // Create a data URL with the media type and base64 encoded image data.
    dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, dataEncoded)

    vid, err := cfg.db.GetVideo(videoID)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "User not authorized", err)
        return
    }

    //tn := thumbnail{
    //    data: data,
    //    mediaType: header.Header.Get("Content-Type"),
    //}

    //videoThumbnails[vid.ID] = tn

    //thumbnailUrl := fmt.Sprintf("/api/thumbnails/%s", vid.ID)
    //vid.ThumbnailURL =  &thumbnailUrl
    vid.ThumbnailURL =  &dataURL

    err = cfg.db.UpdateVideo(vid)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "", err)
        return
    }

    respondWithJSON(w, http.StatusOK, struct {
        Message string `json:"message"`
    }{
        Message: "Thumbnail uploaded successfully",
    })

}
