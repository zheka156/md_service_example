package utils

import "github.com/shopspring/decimal"

func StringToDecimal(amount string) (decimal.Decimal, error) {
	priceDec, err := decimal.NewFromString(amount)
		if err != nil {
			return decimal.Zero, err
		}
	return priceDec, nil
}

func CalculateAmount(quantity decimal.Decimal, price decimal.Decimal) decimal.Decimal {
	return quantity.Mul(price).Round(2)
}