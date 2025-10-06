package server

import (

	"github.com/gofiber/fiber/v2"
	"github.com/zheka156/market_data/internal/utils"
)

func (s *Server) GetStockLastPrice(c *fiber.Ctx) error {
	ticker := c.Params("ticker")

	if !utils.ValidateTicker(ticker) {
		s.log.Sugar().Warnf("Incorrect ticker sent: %s", ticker)
		return c.Status(fiber.StatusBadRequest).SendString("Incorrect ticker sent")
	}

	resp := s.polygonClient.GetLastDatePrices(ticker)
	if resp.Status != "OK" {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	responseMessage := LastPriceResponse{
		Ticker: ticker,
		Last:   resp.Close,
		RequestedDate:  resp.From,
	}
	return c.JSON(responseMessage)
}

type LastPriceResponse struct {
	Ticker string  `json:"ticker"`
	Last   float64 `json:"last"`
	RequestedDate  string  `json:"from"`
}
