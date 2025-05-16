package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) PutObject(c echo.Context) error {
	ctx := c.Request().Context()

	logger := log.Ctx(ctx).With().Str("command", "PutObject").Logger()

	logger.Debug().Msg("PutObject.Start")

	bucketName := c.Param("bucket")
	if bucketName == "" {
		logger.Warn().Msg("Bucket name is missing")
		return c.String(http.StatusBadRequest, "Bucket name is missing")
	}

	logger = logger.With().Str("bucket", bucketName).Logger()

	objectKey := c.Param("*")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}

	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing after trimming prefix")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}

	logger.Debug().Str("bucket", bucketName).Str("key", objectKey).Msg("Parsed parameters")

	repo, err := h.git.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to clone repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to clone repository '%s': %v", bucketName, err))
	}
	logger.Debug().Str("bucket", bucketName).Msg("Repository cloned successfully")

	err = h.git.PutRaw(ctx, repo, objectKey, c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Str("key", objectKey).Msg("Failed to put object into repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to put object '%s': %v", objectKey, err))
	}
	logger.Debug().Str("key", objectKey).Msg("Object put into repository successfully")

	var commitSHA string
	headRef, err := repo.Head()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get repository HEAD for versioning")
		// it's ok to not have a commit SHA
	}

	commitSHA = headRef.Hash().String()
	logger.Debug().Str("commitSHA", commitSHA).Msg("Got commit SHA for version ID")

	c.Response().Header().Set("ETag", fmt.Sprintf(`"%s"`, commitSHA)) // ETag should be quoted

	if commitSHA != "" {
		c.Response().Header().Set("x-amz-version-id", commitSHA)
	}

	logger.Info().Str("bucket", bucketName).Str("key", objectKey).Str("versionId", commitSHA).Msg("PutObject.OK")
	return c.String(http.StatusOK, "Object uploaded successfully. Version ID: "+commitSHA)
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
