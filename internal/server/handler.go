package server

import (
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"github-as-s3/internal/s3"
	"strings"

	"github.com/labstack/echo/v4"
)

// Handler implements the S3API interface.
type Handler struct {
	gh  *github.GitHub
	git *git.Git
}

// NewHandler returns a new Handler instance.
func NewS3Handler(gh *github.GitHub, git *git.Git) S3API {
	return &Handler{
		gh:  gh,
		git: git,
	}
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
