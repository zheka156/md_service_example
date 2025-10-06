package server

import "github.com/gofiber/fiber/v2"

func (s *Server) InitRoutes(router *fiber.App) {
	router.Get("/previousDateQuotes/:ticker", s.GetStockLastPrice)
}
