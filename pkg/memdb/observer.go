package memdb

import (
	"context"

	"github.com/go-kiss/sniper/pkg/log"
	"github.com/go-kiss/sniper/pkg/trace"
	"github.com/go-redis/redis/extra/rediscmd/v8"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// 观察所有 redis 命令执行情况
type observer struct {
	name string
}

func (o observer) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, cmd.FullName())

	ext.Component.Set(span, "memdb")
	ext.DBInstance.Set(span, o.name)
	ext.DBStatement.Set(span, rediscmd.CmdString(cmd))

	return ctx, nil
}

func (o observer) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span := opentracing.SpanFromContext(ctx)
	if err := cmd.Err(); err != nil && err != redis.Nil {
		ext.Error.Set(span, true)
		ext.LogError(span, err)
	}
	span.Finish()

	d := trace.GetDuration(span)
	log.Get(ctx).Debugf("[memdb] %s, cost:%v", rediscmd.CmdString(cmd), d)

	redisDurations.WithLabelValues(
		o.name,
		cmd.FullName(),
	).Observe(d.Seconds())

	return nil
}

func (o observer) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (o observer) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	return nil
}
