package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vnsoft2014/prisma-client-go/test"
)

type cx = context.Context
type Func func(t *testing.T, client *PrismaClient, ctx cx)

func TestTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before []string
		run    Func
	}{{
		name: "transaction",
		run: func(t *testing.T, client *PrismaClient, ctx cx) {
			createUserA := client.User.CreateOne(
				User.Email.Set("a"),
				User.ID.Set("a"),
			).Tx()

			createUserB := client.User.CreateOne(
				User.Email.Set("b"),
				User.ID.Set("b"),
			).Tx()

			if err := client.Prisma.Transaction(createUserA, createUserB).Exec(ctx); err != nil {
				t.Fatal(err)
			}

			// --

			actual, err := client.User.FindMany().Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

			expected := []UserModel{{
				InnerUser: InnerUser{
					ID:    "a",
					Email: "a",
				},
			}, {
				InnerUser: InnerUser{
					ID:    "b",
					Email: "b",
				},
			}}

			assert.Equal(t, expected, actual)
		},
	}, {
		name: "rollback tx",
		// language=GraphQL
		before: []string{`
			mutation {
				result: createOneUser(data: {
					id: "exists",
					email: "email",
				}) {
					id
				}
			}
		`},
		run: func(t *testing.T, client *PrismaClient, ctx cx) {
			// this will fail...
			a := client.User.FindUnique(
				User.ID.Equals("does-not-exist"),
			).Update(
				User.Email.Set("foo"),
			).Tx()

			// ...so this should be roll-backed
			b := client.User.FindUnique(
				User.ID.Equals("exists"),
			).Update(
				User.Email.Set("new"),
			).Tx()

			err := client.Prisma.Transaction(a, b).Exec(ctx)
			assert.Errorf(t, err, "should error")

			// make sure the existing record wasn't touched

			actual, err := client.User.FindMany().Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

			expected := []UserModel{{
				InnerUser: InnerUser{
					ID:    "exists",
					Email: "email",
				},
			}}

			assert.Equal(t, expected, actual)
		},
	}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.RunSerial(t, test.Databases, func(t *testing.T, db test.Database, ctx context.Context) {
				client := NewClient()
				mockDBName := test.Start(t, db, client.Engine, tt.before)
				defer test.End(t, db, client.Engine, mockDBName)
				tt.run(t, client, context.Background())
			})
		})
	}
}
