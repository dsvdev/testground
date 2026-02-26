package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"testground"
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
	conn, err := p.container.Conn(ctx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	var args pgx.NamedArgs
	if len(p.args) > 0 {
		args = p.args[0]
	}

	_, err = conn.Exec(ctx, p.sql, args)
	return err
}
