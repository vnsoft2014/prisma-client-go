package composite

import (
	"context"
	"testing"

	"github.com/vnsoft2014/prisma-client-go/test"
)

type cx = context.Context
type Func func(t *testing.T, client *PrismaClient, ctx cx)

func TestCompositeSelf(t *testing.T) {
	tests := []struct {
		name   string
		before []string
		run    Func
	}{{
		name:   "self unchecked scalar",
		before: nil,
		run: func(t *testing.T, client *PrismaClient, ctx cx) {
			_, err := client.Event.CreateOne(
				Event.ID.Set("event-1"),
			).Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

			_, err = client.Event.CreateOne(
				Event.ID.Set("event-instance-2"),
				Event.PreviousEventID.Set("event-1"),
			).Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

		},
	}, {
		name:   "self link",
		before: nil,
		run: func(t *testing.T, client *PrismaClient, ctx cx) {
			_, err := client.Event.CreateOne(
				Event.ID.Set("event-1"),
			).Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

			_, err = client.Event.CreateOne(
				Event.Previous.Link(
					Event.ID.Equals("event-1"),
				),
			).Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}
		},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.RunSerial(t, []test.Database{test.MySQL, test.PostgreSQL, test.SQLite}, func(t *testing.T, db test.Database, ctx context.Context) {
				client := NewClient()
				mockDBName := test.Start(t, db, client.Engine, tt.before)
				defer test.End(t, db, client.Engine, mockDBName)
				tt.run(t, client, context.Background())
			})
		})
	}
}
