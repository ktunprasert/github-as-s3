package server

import "github.com/labstack/echo/v4"

// API defines the S3-compatible API interface.
type S3API interface {
	CreateBucket(echo.Context) error
	DeleteBucket(echo.Context) error
	ListBuckets(echo.Context) error
	HeadBucket(echo.Context) error

	PutObject(echo.Context) error
	GetObject(echo.Context) error
	ListObjectsV2(echo.Context) error
	DeleteObject(echo.Context) error
	HeadObject(echo.Context) error
}

// RegisterRoutes registers S3-compatible routes with the Echo router.
func RegisterRoutes(e *echo.Echo, api Handler) {
	// Bucket operations
	e.PUT("/:bucket", api.CreateBucket)
	e.DELETE("/:bucket", api.DeleteBucket)
	e.GET("/", api.ListBuckets)
	e.HEAD("/:bucket", api.HeadBucket)

	if api.async {
		e.PUT("/:bucket/*", api.PutObjectAsync)
		e.DELETE("/:bucket/*", api.DeleteObjectAsync)
		e.HEAD("/:bucket/*", api.HeadObjectAsync)
	} else {
		e.PUT("/:bucket/*", api.PutObject)
		e.DELETE("/:bucket/*", api.DeleteObject)
		e.HEAD("/:bucket/*", api.HeadObject)
	}
	e.GET("/:bucket/*", api.GetObject)
	e.GET("/:bucket", api.ListObjectsV2)
	e.GET("/:bucket/", api.ListObjectsV2) // Handles ?list-type=2

	e.Any("/*", api.CatchAllHandler)
}
