package handler

import (
	"context"
	"log/slog"

	apiv1 "github.com/houssemlou/fizz/internal/api/v1"
	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
)

type Service interface {
	Generate(ctx context.Context, p fizzbuzz.Params) ([]string, error)
	Top(ctx context.Context) (*fizzbuzz.TopResult, error)
}

type Handler struct {
	service Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) GetFizzBuzz(ctx context.Context, req apiv1.GetFizzBuzzRequestObject) (apiv1.GetFizzBuzzResponseObject, error) {
	p := fizzbuzz.Params{
		Int1:  req.Params.Int1,
		Int2:  req.Params.Int2,
		Limit: req.Params.Limit,
		Str1:  req.Params.Str1,
		Str2:  req.Params.Str2,
	}
	if err := p.Validate(); err != nil {
		return apiv1.GetFizzBuzz400JSONResponse{BadRequestJSONResponse: apiv1.BadRequestJSONResponse{Error: err.Error()}}, nil
	}
	result, err := h.service.Generate(ctx, p)
	if err != nil {
		return apiv1.GetFizzBuzz500JSONResponse{
			InternalServerErrorJSONResponse: apiv1.InternalServerErrorJSONResponse{Error: err.Error()},
		}, nil
	}
	return apiv1.GetFizzBuzz200JSONResponse{Result: result}, nil
}

func (h *Handler) GetStats(ctx context.Context, _ apiv1.GetStatsRequestObject) (apiv1.GetStatsResponseObject, error) {
	top, err := h.service.Top(ctx)
	if err != nil {
		slog.Error("stats query failed", "error", err)
		return apiv1.GetStats500JSONResponse{
			InternalServerErrorJSONResponse: apiv1.InternalServerErrorJSONResponse{Error: "internal server error"},
		}, nil
	}
	if top == nil {
		msg := "no requests recorded yet"
		return apiv1.GetStats200JSONResponse{Message: &msg}, nil
	}
	hits := top.Hits
	return apiv1.GetStats200JSONResponse{
		Request: &apiv1.FizzBuzzParams{
			Int1:  top.Params.Int1,
			Int2:  top.Params.Int2,
			Limit: top.Params.Limit,
			Str1:  top.Params.Str1,
			Str2:  top.Params.Str2,
		},
		Hits: &hits,
	}, nil
}

func (h *Handler) GetHealth(_ context.Context, _ apiv1.GetHealthRequestObject) (apiv1.GetHealthResponseObject, error) {
	return apiv1.GetHealth200JSONResponse{Status: "ok"}, nil
}
