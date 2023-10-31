package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/fatih/color"
	"github.com/jackc/pgconn"
	"github.com/uptrace/bun"
)

// https://github.com/uptrace/bun/blob/8a4383505d7e954897b23811a412b9cdafaf41eb/LICENSE#L1
// Mostly copied from https://github.com/uptrace/bun/blob/master/extra/bundebug/debug.go
type bunLogger struct {
	logEveryQuery bool
}

func (h *bunLogger) BeforeQuery(
	ctx context.Context, event *bun.QueryEvent,
) context.Context {
	return ctx
}

func (h *bunLogger) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if !h.logEveryQuery {
		if event.Err == nil ||
			errors.Is(event.Err, sql.ErrNoRows) ||
			errors.Is(event.Err, sql.ErrTxDone) ||
			errors.Is(event.Err, context.Canceled) ||
			pgconn.Timeout(event.Err) {
			return
		}
	}

	now := time.Now()
	dur := now.Sub(event.StartTime)

	args := []interface{}{
		"[bun]",
		now.Format(" 15:04:05.000 "),
		formatOperation(event),
		fmt.Sprintf(" %10s ", dur.Round(time.Microsecond)),
		event.Query,
	}

	if event.Err != nil {
		typ := reflect.TypeOf(event.Err).String()
		args = append(args,
			"\t",
			color.New(color.BgRed).Sprintf(" %s ", typ+": "+event.Err.Error()),
		)

		log.Error(args...)
	} else {
		log.Debug(args...)
	}
}

func formatOperation(event *bun.QueryEvent) string {
	operation := event.Operation()
	return operationColor(operation).Sprintf(" %-16s ", operation)
}

func operationColor(operation string) *color.Color {
	switch operation {
	case "SELECT":
		return color.New(color.BgGreen, color.FgHiWhite)
	case "INSERT":
		return color.New(color.BgBlue, color.FgHiWhite)
	case "UPDATE":
		return color.New(color.BgYellow, color.FgHiBlack)
	case "DELETE":
		return color.New(color.BgMagenta, color.FgHiWhite)
	default:
		return color.New(color.BgWhite, color.FgHiBlack)
	}
}
