package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/dsvdev/testground"
)

type execPrecondition struct {
	container *Container
	sql       string
	args      []pgx.NamedArgs
}

func (c *Container) Exec(sql string, args ...pgx.NamedArgs) testground.Precondition {
	return &execPrecondition{container: c, sql: sql, args: args}
}

func (p *execPrecondition) Apply(ctx context.Context, t *testing.T) error {
	pool, err := p.container.Pool(ctx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	var namedArgs pgx.NamedArgs
	if len(p.args) > 0 {
		namedArgs = p.args[0]
	}

	_, err = pool.Exec(ctx, p.sql, namedArgs)
	return err
}
