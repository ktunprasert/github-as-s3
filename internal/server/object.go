package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

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
