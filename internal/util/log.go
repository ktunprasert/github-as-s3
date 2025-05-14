package util

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func LogCtx(ctx context.Context, contextParent string) *zerolog.Logger {
	slog := log.Ctx(ctx)
	if !slog.Info().Enabled() {
		v := log.With().Str("contextParent", contextParent).Logger()
		slog = &v
	}

	return slog
}
