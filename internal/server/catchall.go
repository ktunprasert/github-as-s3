package server

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) CatchAllHandler(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("path", c.Path()).Logger()
	logger.Info().Msg("CatchAllHandler invoked")

	logger.Info().Str("method", c.Request().Method).Msg("Method")
	logger.Info().Any("header", c.Request().Header).Msg("Request")
	logger.Info().Any("query", c.QueryParams()).Msg("Query")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Msg("Error reading request body")
	} else {
		logger.Info().Str("body", string(body)).Msg("Request Body")
	}

	// This is a catch-all handler for any unmatched routes.
	// It can be used to return a 404 Not Found or a custom error message.
	return c.String(http.StatusNotImplemented, "Not Implemented")
}
