package server

import (
	"errors"
	"fmt"
	"github-as-s3/internal/git"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) PutObjectAsync(c echo.Context) error {
	ctx := c.Request().Context()

	logger := log.Ctx(ctx).With().Str("command", "PutObject").Bool("async", true).Logger()

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

	// async difference starts
	repo, err := h.gitasync.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		return c.String(http.StatusInternalServerError, "Failed to clone repository")
	}
	logger.Debug().Msg("Cloned repository")

	err = h.gitasync.PutRaw(ctx, repo, bucketName, objectKey, c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to put object")
		return c.String(http.StatusInternalServerError, "Failed to put object")
	}

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

	logger.Info().Str("versionId", commitSHA).Msg("PutObject.OK")
	return c.String(http.StatusOK, "")
}

func (h *Handler) DeleteObjectAsync(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("command", "DeleteObject").Logger()

	logger.Debug().Msg("DeleteObject.Start")

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
	logger = logger.With().Str("key", objectKey).Logger()

	logger.Debug().Msg("Parsed parameters")

	repo, err := h.gitasync.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		// could make this stricter if we wanted: return c.String(http.StatusNotFound, fmt.Sprintf("Bucket (repository) '%s' not found: %v", bucketName, err))
		c.Response().Header().Set("x-amz-delete-marker", "true")
		logger.Info().Msg("Bucket not found, but DeleteObject is idempotent. Responding with 204 No Content.")
		return c.NoContent(http.StatusNoContent)
	}
	logger.Debug().Msg("Repository cloned successfully")

	err = h.gitasync.Delete(ctx, repo, bucketName, objectKey)
	if err != nil {
		if errors.Is(err, git.ErrFileNotExists) {
			logger.Info().Err(err).Msg("Object not found in repository, delete is idempotent.")
			// S3 returns 204 No Content if the object to be deleted is not found.
		} else {
			logger.Error().Err(err).Msg("Failed to delete object from repository")
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to delete object '%s': %v", objectKey, err))
		}
	} else {
		logger.Debug().Msg("Object deleted from repository successfully")
	}

	var deleteMarkerVersionID string
	headRef, headErr := repo.Head()
	if headErr != nil {
		logger.Warn().Err(headErr).Msg("Failed to get repository HEAD for versioning after delete.")
	} else if headRef != nil {
		deleteMarkerVersionID = headRef.Hash().String()
		logger.Debug().Str("deleteMarkerVersionID", deleteMarkerVersionID).Msg("Got commit SHA for delete marker version ID")
	} else {
		logger.Warn().Msg("repo.Head() returned nil ref without error after delete.")
	}

	c.Response().Header().Set("x-amz-delete-marker", "true")
	if deleteMarkerVersionID != "" {
		// For versioned buckets, S3 returns x-amz-version-id for the delete marker
		c.Response().Header().Set("x-amz-version-id", deleteMarkerVersionID)
	}

	logger.Info().Str("key", objectKey).Str("bucket", bucketName).Str("versionId", deleteMarkerVersionID).Msg("DeleteObject.OK")
	return c.NoContent(http.StatusNoContent)
}
