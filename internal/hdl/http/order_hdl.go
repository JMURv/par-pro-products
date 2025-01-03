package http

import (
	"errors"
	ctrl "github.com/JMURv/par-pro/products/internal/ctrl"
	mid "github.com/JMURv/par-pro/products/internal/hdl/http/middleware"
	metrics "github.com/JMURv/par-pro/products/internal/metrics/prometheus"
	"github.com/JMURv/par-pro/products/internal/validation"
	"github.com/JMURv/par-pro/products/pkg/consts"
	"github.com/JMURv/par-pro/products/pkg/model"
	utils "github.com/JMURv/par-pro/products/pkg/utils/http"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func RegisterOrderRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc(
		"/api/order/me", mid.ApplyMiddleware(
			h.listUserOrders, mid.MethodNotAllowed(http.MethodGet), h.authMiddleware,
		),
	)
	mux.HandleFunc(
		"/api/order", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mid.ApplyMiddleware(h.listOrders, h.authMiddleware)(w, r)
			case http.MethodPost:
				mid.ApplyMiddleware(h.createOrder)(w, r)
			default:
				utils.ErrResponse(w, http.StatusMethodNotAllowed, mid.ErrMethodNotAllowed)
			}
		},
	)

	mux.HandleFunc(
		"/api/order/", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mid.ApplyMiddleware(h.getOrder, h.authMiddleware)(w, r)
			case http.MethodPut:
				mid.ApplyMiddleware(h.updateOrder, h.authMiddleware)(w, r)
			case http.MethodDelete:
				mid.ApplyMiddleware(h.cancelOrder, h.authMiddleware)(w, r)
			default:
				utils.ErrResponse(w, http.StatusMethodNotAllowed, mid.ErrMethodNotAllowed)
			}
		},
	)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "orders.ListOrders.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = consts.DefaultPage
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = consts.DefaultPageSize
	}

	sort := r.URL.Query().Get("sort")
	filters := utils.ParseFiltersByURL(r)

	res, err := h.ctrl.ListOrders(r.Context(), page, size, filters, sort)
	if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to get orders", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessPaginatedResponse(w, c, res)
}

func (h *Handler) listUserOrders(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "orders.listUserOrders.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	uid, err := uuid.Parse(r.Context().Value("uid").(string))
	if err != nil {
		c = http.StatusUnauthorized
		zap.L().Debug("Invalid token", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = consts.DefaultPage
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = consts.DefaultPageSize
	}

	res, err := h.ctrl.ListUserOrders(r.Context(), uid, page, size)
	if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to get orders", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessPaginatedResponse(w, c, res)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "orders.getOrder.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	orderID, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/order/"), 10, 64)
	if err != nil {
		c = http.StatusBadRequest
		utils.ErrResponse(w, c, err)
		return
	}

	res, err := h.ctrl.GetOrder(r.Context(), orderID)
	if err != nil && errors.Is(err, ctrl.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("failed to get order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to get order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var err error
	s, c := time.Now(), http.StatusCreated
	const op = "orders.createOrder.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	req := &model.Order{}
	if err = json.NewDecoder(r.Body).Decode(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to decode request", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	if err = validation.Order(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to validate obj", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	uid := uuid.Nil
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != authHeader {
			claims, _ := h.sso.ParseClaims(r.Context(), token)
			uid, err = uuid.Parse(claims)
			if err != nil {
				zap.L().Debug("Error parse user UUID", zap.Error(err), zap.String("op", op))
				c = http.StatusBadRequest
				utils.ErrResponse(w, c, ctrl.ErrInternalError)
				return
			}
		}
	}

	if uid == uuid.Nil {
		genPass := uuid.NewString()
		userID, err := h.sso.CreateUser(r.Context(), req.FIO, req.Email, genPass)
		if err != nil {
			zap.L().Debug("Error create user", zap.Error(err), zap.String("op", op))
			c = http.StatusBadRequest
			utils.ErrResponse(w, c, err)
			return
		}

		uid, err = uuid.Parse(userID)
		if err != nil {
			zap.L().Debug("Error parse user UUID", zap.Error(err), zap.String("op", op))
			utils.ErrResponse(w, c, ctrl.ErrInternalError)
			return
		}
	}

	res, err := h.ctrl.CreateOrder(r.Context(), uid, req)
	if err != nil && errors.Is(err, ctrl.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("failed to create order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil && errors.Is(err, ctrl.ErrAlreadyExists) {
		c = http.StatusConflict
		zap.L().Debug("failed to create order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to create order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) updateOrder(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "orders.updateOrder.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	req := &model.Order{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to decode request", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	if err := validation.Order(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to validate obj", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	orderID, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/order/"), 10, 64)
	if err != nil {
		c = http.StatusBadRequest
		utils.ErrResponse(w, c, err)
		return
	}

	err = h.ctrl.UpdateOrder(r.Context(), orderID, req)
	if err != nil && errors.Is(err, ctrl.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("failed to update order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to update order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessResponse(w, c, "OK")
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "orders.cancelOrder.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	orderID, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/order/"), 10, 64)
	if err != nil {
		c = http.StatusBadRequest
		utils.ErrResponse(w, c, err)
		return
	}

	err = h.ctrl.CancelOrder(r.Context(), orderID)
	if err != nil && errors.Is(err, ctrl.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("failed to cancel order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to cancel order", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, ctrl.ErrInternalError)
		return
	}

	utils.SuccessResponse(w, c, "OK")
}
