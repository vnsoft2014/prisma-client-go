package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/shopspring/decimal"
)

const RFC3339Milli = "2006-01-02T15:04:05.999Z07:00"

type BatchResult struct {
	Count int `json:"count"`
}

// DateTime is a type alias for time.Time
type DateTime = time.Time

// Decimal points to github.com/shopspring/decimal.Decimal, as Go does not have a native decimal type
type Decimal = decimal.Decimal

// Bytes is a type alias for []byte
type Bytes = []byte

// BigInt is a type alias for int64
type BigInt int64

// UnmarshalJSON converts the Prisma QE value of string to int64
func (m *BigInt) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("BigInt: UnmarshalJSON on nil pointer")
	}
	str, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("BigInt: unquote: %w", err)
	}
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("BigInt: UnmarshalJSON error: %w", err)
	}
	*m = BigInt(i)
	return nil
}

// MarshalJSON converts the input to a Prisma value
func (m *BigInt) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%d\"", *m)), nil
}

// JSON is a new type which implements the correct internal prisma (un)marshaller
type JSON map[string]interface{}

func MarshalJSON(b JSON) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		byteData, err := json.Marshal(b)
		if err != nil {
			log.Printf("FAIL WHILE MARSHAL JSON %v\n", string(byteData))
		}
		_, err = w.Write(byteData)
		if err != nil {
			log.Printf("FAIL WHILE WRITE DATA %v\n", string(byteData))
		}
	})
}

func UnmarshalJSON(v interface{}) (JSON, error) {
	byteData, err := json.Marshal(v)
	if err != nil {
		return JSON{}, fmt.Errorf("FAIL WHILE MARSHAL SCHEME")
	}
	tmp := make(map[string]interface{})
	err = json.Unmarshal(byteData, &tmp)
	if err != nil {
		return JSON{}, fmt.Errorf("FAIL WHILE UNMARSHAL SCHEME")
	}
	return tmp, nil
}
