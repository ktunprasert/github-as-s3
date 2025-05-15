package server

import "github.com/labstack/echo/v4"

// API defines the S3-compatible API interface.
type S3API interface {
	CreateBucket(echo.Context) error
	DeleteBucket(echo.Context) error
	ListBuckets(echo.Context) error
	PutObject(echo.Context) error
	GetObject(echo.Context) error
	ListObjectsV2(echo.Context) error
	DeleteObject(echo.Context) error
}

// RegisterRoutes registers S3-compatible routes with the Echo router.
func RegisterRoutes(e *echo.Echo, api S3API) {
	// Bucket operations
	e.PUT("/:bucket", api.CreateBucket)
	e.DELETE("/:bucket", api.DeleteBucket)
	e.GET("/", api.ListBuckets)

	// Object operations
	e.PUT("/:bucket/*", api.PutObject)
	e.GET("/:bucket/*", api.GetObject)
	e.GET("/:bucket", api.ListObjectsV2) // Handles ?list-type=2
	e.DELETE("/:bucket/*", api.DeleteObject)
}
