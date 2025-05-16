package server

import (
	"encoding/xml"
	"errors"
	"github-as-s3/internal/github"
	"github-as-s3/internal/s3"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) CreateBucket(c echo.Context) error {
	bucketName := c.Param("bucket")
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("bucket", bucketName).Str("command", "CreateBucket").Logger()

	if !s3.IsValidBucketName(bucketName) {
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

	// we dont care about body content tbh
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html#API_CreateBucket_RequestBody
	// if c.Request().ContentLength > 0 {
	// 	bodyBytes, err := io.ReadAll(c.Request().Body)
	// 	if err != nil {
	// 		logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to read request body for CreateBucketConfiguration")
	// 		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error. Please try again.", bucketName)
	// 	}
	// 	defer c.Request().Body.Close()

	// 	if len(bodyBytes) > 0 {
	// 		var config CreateBucketConfiguration
	// 		if err := xml.Unmarshal(bodyBytes, &config); err != nil {
	// 			logger.Warn().Err(err).Str("bucket", bucketName).Msg("Malformed XML in CreateBucketConfiguration")
	// 			return h.s3ErrorResponse(c, http.StatusBadRequest, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema.", bucketName)
	// 		}
	// 		logger.Info().Str("bucket", bucketName).Str("locationConstraint", config.LocationConstraint).Msg("Parsed CreateBucketConfiguration (LocationConstraint will be ignored)")
	// 	}
	// }

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
	bucketName := c.Param("bucket")
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("bucket", bucketName).Str("command", "DeleteBucket").Logger()

	logger.Debug().Str("bucket", bucketName).Msg("Processing DeleteBucket request")

	err := h.gh.DeleteRepo(ctx, bucketName)
	if err != nil {
		if errors.Is(err, github.ErrRepoNotFound) { // Assuming github.ErrRepoNotFound exists
			logger.Warn().Str("bucket", bucketName).Msg("Attempted to delete a bucket that does not exist (repository not found)")
			return h.s3ErrorResponse(c, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.", bucketName)
		}

		logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to delete GitHub repository")
		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error deleting the repository. Please try again.", bucketName)
	}

	logger.Info().Str("bucket", bucketName).Msg("Bucket deleted successfully (GitHub repository deleted)")
	return c.NoContent(http.StatusNoContent)
}

const defaultMaxBuckets = 25
const githubMaxPerPage = 100

func (h *Handler) ListBuckets(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("command", "ListBuckets").Logger()

	continuationTokenStr := c.QueryParam("continuation-token")
	maxBucketsStr := c.QueryParam("max-buckets")
	prefix := c.QueryParam("prefix")

	logger.Debug().
		Str("continuation-token", continuationTokenStr).
		Str("max-buckets", maxBucketsStr).
		Str("prefix", prefix).
		Msg("Processing ListBuckets request")

	page := 1
	if continuationTokenStr != "" {
		parsedPage, err := strconv.Atoi(continuationTokenStr)
		if err == nil && parsedPage > 0 {
			page = parsedPage
		} else {
			logger.Warn().Str("continuation-token", continuationTokenStr).Msg("Invalid continuation-token, using default page 1")
		}
	}

	perPage := defaultMaxBuckets
	if maxBucketsStr != "" {
		parsedMax, err := strconv.Atoi(maxBucketsStr)
		if err == nil && parsedMax > 0 {
			perPage = parsedMax
			if perPage > githubMaxPerPage {
				logger.Warn().Int("requested_max_buckets", perPage).Int("capped_at", githubMaxPerPage).Msg("max-buckets capped")
				perPage = githubMaxPerPage
			}
		} else {
			logger.Warn().Str("max-buckets", maxBucketsStr).Msg("Invalid max-buckets, using default")
		}
	}

	ghReposPage, nextPageFromGH, incompleteResults, err := h.gh.ListRepos(ctx, page, perPage, prefix)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list GitHub repositories")
		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error listing repositories.", "")
	}

	s3ApiBuckets := make([]s3.Bucket, 0, len(ghReposPage))
	for _, repo := range ghReposPage {
		if repo.Name == nil || repo.CreatedAt == nil {
			logger.Warn().Str("repo_id", repo.GetNodeID()).Msg("Skipping repository with nil name or creation date during ListBuckets")
			continue
		}
		s3ApiBuckets = append(s3ApiBuckets, s3.Bucket{
			Name:         repo.GetName(),
			CreationDate: repo.GetCreatedAt().Time.UTC().Format(time.RFC3339),
		})
	}

	owner := h.gh.GetOwner()

	s3Owner := s3.Owner{
		ID:          owner,
		DisplayName: owner,
	}

	result := s3.ListAllMyBucketsResult{
		Owner:   s3Owner,
		Buckets: s3ApiBuckets,
	}

	if incompleteResults {
		result.IsTruncated = true
		result.NextContinuationToken = nextPageFromGH
	}

	xmlBytes, err := xml.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal ListBuckets response to XML")
		return h.s3ErrorResponse(c, http.StatusInternalServerError, "InternalError", "We encountered an internal error preparing the response.", "")
	}

	finalXML := xml.Header + string(xmlBytes)

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.String(http.StatusOK, finalXML)
}
