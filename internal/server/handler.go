package server

import (
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"net/http"

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

func (h *Handler) CreateBucket(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "CreateBucket not implemented")
}

func (h *Handler) DeleteBucket(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "DeleteBucket not implemented")
}

func (h *Handler) ListBuckets(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "ListBuckets not implemented")
}

func (h *Handler) PutObject(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "PutObject not implemented")
}

func (h *Handler) GetObject(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "GetObject not implemented")
}

func (h *Handler) ListObjectsV2(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "ListObjectsV2 not implemented")
}

func (h *Handler) DeleteObject(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "DeleteObject not implemented")
}
