package server

import (
	"errors"
	"github-as-s3/internal/github"
	"github-as-s3/internal/s3"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) CreateBucket(c echo.Context) error {
	bucketName := c.Param("bucket")
	ctx := c.Request().Context()
	logger := log.Ctx(ctx)

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
	return c.String(http.StatusNotImplemented, "DeleteBucket not implemented")
}

func (h *Handler) ListBuckets(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "ListBuckets not implemented")
}
