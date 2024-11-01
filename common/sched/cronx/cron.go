package cronx

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/robfig/cron/v3"
	"strings"
)

var C *cron.Cron

type WrappedLogger struct {
}

func (WrappedLogger) Info(msg string, keysAndValues ...any) {
	logger.Infof(formatString(len(keysAndValues)), append([]any{msg}, keysAndValues...)...)
}

func (l WrappedLogger) Error(err error, msg string, keysAndValues ...any) {
	logger.Errorf(formatString(len(keysAndValues)+2), append([]any{msg, "error", err}, keysAndValues...)...)
}

// formatString copy from cron Logger
func formatString(numKeysAndValues int) string {
	var sb strings.Builder
	sb.WriteString("%s")
	if numKeysAndValues > 0 {
		sb.WriteString(", ")
	}
	for i := 0; i < numKeysAndValues/2; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("%v=%v")
	}
	return sb.String()
}

func Setup() {
	C = cron.New(cron.WithLogger(WrappedLogger{}),
		cron.WithParser(cron.NewParser(
			cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)))
	C.Start()
}
