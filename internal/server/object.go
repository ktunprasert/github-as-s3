package server

import (
	"errors"
	"fmt"
	"github-as-s3/internal/git"
	"github-as-s3/internal/s3"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

	logger = logger.With().Str("bucket", bucketName).Str("key", objectKey).Logger()
	logger.Debug().Msg("Parsed parameters")

	repo, err := h.git.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Str("bucket", bucketName).Msg("Failed to clone repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to clone repository '%s': %v", bucketName, err))
	}
	logger.Debug().Msg("Repository cloned successfully")

	err = h.git.PutRaw(ctx, repo, objectKey, c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Str("key", objectKey).Msg("Failed to put object into repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to put object '%s': %v", objectKey, err))
	}
	logger.Debug().Msg("Object put into repository successfully")

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

func (h *Handler) GetObject(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("command", "GetObject").Logger()

	logger.Debug().Msg("GetObject.Start")

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

	repo, err := h.git.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		return c.String(http.StatusNotFound, fmt.Sprintf("Bucket (repository) '%s' not found: %v", bucketName, err))
	}
	logger.Debug().Msg("Repository cloned successfully")

	objectData, fileInfo, err := h.git.Get(ctx, repo, objectKey)
	if err != nil {
		if errors.Is(err, git.ErrFileNotExists) {
			logger.Warn().Err(err).Msg("Object not found in repository")
			return c.String(http.StatusNotFound, fmt.Sprintf("Object '%s' not found in bucket '%s'", objectKey, bucketName))
		}
		logger.Error().Err(err).Msg("Failed to get object from repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get object '%s': %v", objectKey, err))
	}
	logger.Debug().Msg("Object retrieved successfully")

	var commitSHA string
	headRef, err := repo.Head()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get repository HEAD for versioning. ETag/versionId may be affected.")
	} else if headRef != nil {
		commitSHA = headRef.Hash().String()
		logger.Debug().Str("commitSHA", commitSHA).Msg("Got commit SHA for version ID/ETag")
	} else {
		logger.Warn().Msg("repo.Head() returned nil ref without error. ETag/versionId may be affected.")
	}

	if commitSHA != "" {
		c.Response().Header().Set("ETag", fmt.Sprintf(`"%s"`, commitSHA))
		c.Response().Header().Set("x-amz-version-id", commitSHA)
	}

	if fileInfo != nil {
		c.Response().Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
		c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	} else {
		// Fallback if fileInfo is somehow nil, though git.Get should provide it.
		c.Response().Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(objectData)))
	}

	contentType := http.DetectContentType(objectData)
	c.Response().Header().Set("Content-Type", contentType)

	logger.Info().Str("key", objectKey).Str("bucket", bucketName).Str("versionId", commitSHA).Msg("GetObject.OK")
	return c.Blob(http.StatusOK, contentType, objectData)
}

func (h *Handler) DeleteObject(c echo.Context) error {
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

	repo, err := h.git.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		// could make this stricter if we wanted: return c.String(http.StatusNotFound, fmt.Sprintf("Bucket (repository) '%s' not found: %v", bucketName, err))
		c.Response().Header().Set("x-amz-delete-marker", "true")
		logger.Info().Msg("Bucket not found, but DeleteObject is idempotent. Responding with 204 No Content.")
		return c.NoContent(http.StatusNoContent)
	}
	logger.Debug().Msg("Repository cloned successfully")

	err = h.git.Delete(ctx, repo, objectKey)
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

func (h *Handler) ListObjectsV2(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("command", "ListObjectsV2").Logger()

	bucketName := c.Param("bucket")
	if bucketName == "" {
		logger.Warn().Msg("Bucket name is missing")
		return c.String(http.StatusBadRequest, "Bucket name is missing")
	}
	logger = logger.With().Str("bucket", bucketName).Logger()

	// Parse query parameters for echoing back, but not for limiting results
	requestPrefix := c.QueryParam("prefix")
	requestDelimiter := c.QueryParam("delimiter")
	requestMaxKeysStr := c.QueryParam("max-keys")
	requestContinuationToken := c.QueryParam("continuation-token") // Will be echoed
	requestStartAfter := c.QueryParam("start-after")               // Will be echoed

	maxKeysForResponse := 1000 // Default MaxKeys for S3 response field
	if requestMaxKeysStr != "" {
		parsedMaxKeys, err := strconv.Atoi(requestMaxKeysStr)
		if err == nil && parsedMaxKeys >= 0 { // S3 allows 0 for MaxKeys
			maxKeysForResponse = parsedMaxKeys
		} else {
			logger.Warn().Str("max-keys", requestMaxKeysStr).Msg("Invalid max-keys value, using default for response field.")
		}
	}

	logger.Debug().
		Str("prefix", requestPrefix).
		Str("delimiter", requestDelimiter).
		Int("maxKeys (for_response_echo)", maxKeysForResponse).
		Str("continuationToken (for_response_echo)", requestContinuationToken).
		Str("startAfter (for_response_echo)", requestStartAfter).
		Msg("ListObjectsV2.Start - returning all results")

	repo, err := h.git.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		return c.String(http.StatusNotFound, fmt.Sprintf("Bucket '%s' not found", bucketName))
	}

	allFilesInfo, err := h.git.List(ctx, repo)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list objects from repository")
		return c.String(http.StatusInternalServerError, "Failed to list objects")
	}

	var sortedKeys []string
	for k := range allFilesInfo {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var contents []s3.ContentsType
	commonPrefixesMap := make(map[string]struct{})

	for _, key := range sortedKeys {
		fileInfo := allFilesInfo[key] // Get the os.FileInfo for the key

		if requestPrefix != "" && !strings.HasPrefix(key, requestPrefix) {
			continue
		}

		if requestDelimiter != "" {
			keyRelativeToPrefix := strings.TrimPrefix(key, requestPrefix)
			delimiterIndex := strings.Index(keyRelativeToPrefix, requestDelimiter)

			if delimiterIndex != -1 {
				// This key contributes to a common prefix
				commonPrefix := requestPrefix + keyRelativeToPrefix[:delimiterIndex+len(requestDelimiter)]
				commonPrefixesMap[commonPrefix] = struct{}{}
			} else {
				// This key is an object
				contents = append(contents, s3.ContentsType{
					Key:          key,
					LastModified: fileInfo.ModTime().UTC().Format("2006-01-02T15:04:05.000Z"),
					Size:         fileInfo.Size(),
					StorageClass: "STANDARD",
					// ETag is omitted as per simplification
				})
			}
		} else {
			// No delimiter, so everything matching the prefix is an object
			contents = append(contents, s3.ContentsType{
				Key:          key,
				LastModified: fileInfo.ModTime().UTC().Format("2006-01-02T15:04:05.000Z"),
				Size:         fileInfo.Size(),
				StorageClass: "STANDARD",
				// ETag is omitted
			})
		}
	}

	var commonPrefixesList []s3.CommonPrefixType
	for cp := range commonPrefixesMap {
		commonPrefixesList = append(commonPrefixesList, s3.CommonPrefixType{Prefix: cp})
	}
	sort.Slice(commonPrefixesList, func(i, j int) bool {
		return commonPrefixesList[i].Prefix < commonPrefixesList[j].Prefix
	})

	result := s3.ListBucketResult{
		Xmlns:             "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:              bucketName,
		Prefix:            requestPrefix,
		Delimiter:         requestDelimiter,
		MaxKeys:           maxKeysForResponse, // Echoing back the parsed or default MaxKeys
		IsTruncated:       false,              // Always false as we return all results
		Contents:          contents,
		CommonPrefixes:    commonPrefixesList,
		KeyCount:          len(contents) + len(commonPrefixesList),
		ContinuationToken: requestContinuationToken, // Echo back if provided
		StartAfter:        requestStartAfter,        // Echo back if provided
		// NextContinuationToken is omitted (or empty string) as IsTruncated is false
	}

	logger.Info().
		Int("returnedContents", len(result.Contents)).
		Int("returnedCommonPrefixes", len(result.CommonPrefixes)).
		Bool("isTruncated", result.IsTruncated).
		Msg("ListObjectsV2.OK - all results returned")

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(http.StatusOK, result)
}
