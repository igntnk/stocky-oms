package repository

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"math/big"
	"strconv"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrEmptyOrder        = errors.New("order must contain at least one product")
	ErrInvalidOrderTotal = errors.New("order total doesn't match products sum")
)

func NumericToFloat64(num pgtype.Numeric) (float64, error) {
	if !num.Valid {
		return 0, fmt.Errorf("numeric value is NULL")
	}

	// Convert Numeric to big.Float
	bf := new(big.Float)
	bf.SetInt(num.Int)

	// Apply the scale (decimal places)
	if num.Exp != 0 {
		scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-num.Exp)), nil))
		bf.Mul(bf, scale)
	}

	// Convert to float64
	f64, accuracy := bf.Float64()
	if accuracy == big.Below || accuracy == big.Above {
		// Indicates potential loss of precision
		return f64, fmt.Errorf("potential precision loss when converting to float64")
	}

	return f64, nil
}

func Float64ToNumericWithPrecision(f float64, precision int) (pgtype.Numeric, error) {
	str := strconv.FormatFloat(f, 'f', precision, 64)

	var num pgtype.Numeric
	err := num.Scan(str)
	if err != nil {
		return pgtype.Numeric{}, err
	}

	return num, nil
}
