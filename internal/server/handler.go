package server

import (
	"errors"
	"fmt"
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"github-as-s3/internal/s3"
	"mime"
	"net/http"
	"path"
	"strings"

	ghlib "github.com/google/go-github/v72/github"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// Handler implements the S3API interface.
type Handler struct {
	gh       *github.GitHub
	git      *git.Git
	gitasync *git.GitAsync
	async    bool
}

// NewHandler returns a new Handler instance.
func NewS3Handler(gh *github.GitHub, g *git.Git, async bool) Handler {
	return Handler{
		gh:       gh,
		git:      g,
		gitasync: git.NewGitAsync(g),
		async:    async,
	}
}

func (h *Handler) HeadBucket(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "Not Implemented")
}

func (h *Handler) HeadObject(c echo.Context) error {
	bucketName := c.Param("bucket")
	objectKey := c.Param("*")
	versionID := c.QueryParam("versionId")

	logger := log.Ctx(c.Request().Context()).With().
		Str("bucket", bucketName).
		Str("object", objectKey).
		Str("versionId", versionID).
		Str("command", "HeadObject").
		Logger()

	logger.Debug().Msg("HeadObject request received")

	fileContent, _, lastModified, err := h.gh.Head(c.Request().Context(), bucketName, objectKey, versionID)

	if err != nil {
		var ghErrResp *ghlib.ErrorResponse
		if errors.As(err, &ghErrResp) && ghErrResp.Response != nil && ghErrResp.Response.StatusCode == http.StatusNotFound {
			return h.s3ErrorResponse(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", objectKey)
		}

		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "An internal error occurred while trying to head the object.", objectKey)
	}

	if fileContent == nil || (fileContent.GetType() != "file" && fileContent.GetType() != "symlink") {
		return h.s3ErrorResponse(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", objectKey)
	}

	logger.Debug().
		Str("file_sha", *fileContent.SHA).
		Int("size", *fileContent.Size).
		Str("type", *fileContent.Type).
		Time("last_modified", *lastModified).
		Msg("Object found, setting headers")

	c.Response().Header().Set("ETag", fmt.Sprintf("\"%s\"", *fileContent.SHA))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", *fileContent.Size))

	c.Response().Header().Set("x-amz-version-id", *fileContent.SHA)

	contentType := mime.TypeByExtension(path.Ext(objectKey))
	if contentType == "" {
		contentType = "application/octet-stream" // S3 default
	}
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Accept-Ranges", "bytes") // Common for S3 objects
	c.Response().Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))

	return c.NoContent(http.StatusOK)
}

func (h *Handler) s3ErrorResponse(c echo.Context, httpStatus int, s3ErrorCode, message, resourceName string) error {
	errResp := s3.S3Error{
		Code:      s3ErrorCode,
		Message:   message,
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		HostID:    "github-as-s3",
	}

	if strings.Contains(strings.ToLower(s3ErrorCode), "bucket") || (resourceName != "" && (s3ErrorCode == "NoSuchKey" || s3ErrorCode == "NoSuchBucket")) {
		errResp.BucketName = resourceName
	} else if resourceName != "" {
		errResp.Resource = resourceName
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(httpStatus, errResp)
}
