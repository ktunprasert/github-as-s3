package server

import (
	"encoding/xml"
	"errors"
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// S3Error represents the XML structure for S3 error responses.
type S3Error struct {
	XMLName    xml.Name `xml:"Error"`
	Code       string   `xml:"Code"`
	Message    string   `xml:"Message"`
	BucketName string   `xml:"BucketName,omitempty"`
	Resource   string   `xml:"Resource,omitempty"`
	RequestID  string   `xml:"RequestId"`
	HostID     string   `xml:"HostId"` // Placeholder HostID
}

type CreateBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint,omitempty"`
}

var (
	bucketNameRegex      = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	ipAddressRegex       = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	disallowedPrefixes   = []string{"xn--"}
	disallowedSuffixes   = []string{"-s3alias", "--ol-s3"}
	disallowedSubstrings = []string{"..", ".-", "-."}
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
	bucketName := c.Param("bucket")
	ctx := c.Request().Context()
	logger := log.Ctx(ctx)

	if !isValidBucketName(bucketName) {
		logger.Warn().Str("bucket", bucketName).Msg("Invalid bucket name format")
		return h.s3ErrorResponse(c, http.StatusBadRequest, "InvalidBucketName", "The specified bucket is not valid.", bucketName)
	}

	var isPrivate bool
	aclHeader := c.Request().Header.Get("X-Amz-Acl")
	logger.Debug().Str("bucket", bucketName).Str("x-amz-acl", aclHeader).Msg("Processing CreateBucket request")

	switch aclHeader {
	case "public-read", "public-read-write":
		isPrivate = false
	case "private", "":
		isPrivate = true
	default:
		logger.Warn().Str("bucket", bucketName).Str("acl", aclHeader).Msg("Unsupported ACL value received, defaulting to private.")
		isPrivate = true
	}

	if c.Request().ContentLength > 0 {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to read request body for CreateBucketConfiguration")
			return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error. Please try again.", bucketName)
		}
		defer c.Request().Body.Close()

		if len(bodyBytes) > 0 {
			var config CreateBucketConfiguration
			if err := xml.Unmarshal(bodyBytes, &config); err != nil {
				logger.Warn().Err(err).Str("bucket", bucketName).Msg("Malformed XML in CreateBucketConfiguration")
				return h.s3ErrorResponse(c, http.StatusBadRequest, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema.", bucketName)
			}
			logger.Info().Str("bucket", bucketName).Str("locationConstraint", config.LocationConstraint).Msg("Parsed CreateBucketConfiguration (LocationConstraint will be ignored)")
		}
	}

	err := h.gh.CreateRepo(ctx, bucketName, isPrivate)
	if err != nil {
		if errors.Is(err, github.ErrRepoAlreadyExists) { // Replace with actual error check
			logger.Warn().Str("bucket", bucketName).Msg("Attempted to create a bucket that already exists (repository exists)")
			return h.s3ErrorResponse(c, http.StatusConflict, "BucketAlreadyOwnedByYou", "Your previous request to create the named bucket succeeded and you already own it.", bucketName)
		}

		logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to create GitHub repository")
		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error creating the repository. Please try again.", bucketName)
	}

	c.Response().Header().Set("Location", "/"+bucketName)
	logger.Info().Str("bucket", bucketName).Bool("isPrivate", isPrivate).Msg("Bucket created successfully (GitHub repository created)")
	return c.NoContent(http.StatusOK)
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

func (h *Handler) s3ErrorResponse(c echo.Context, httpStatus int, s3ErrorCode, message, resourceName string) error {
	errResp := S3Error{
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

func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if !bucketNameRegex.MatchString(name) {
		return false
	}
	if ipAddressRegex.MatchString(name) {
		return false
	}
	for _, prefix := range disallowedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	for _, suffix := range disallowedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	for _, sub := range disallowedSubstrings {
		if strings.Contains(name, sub) {
			return false
		}
	}
	return true
}
