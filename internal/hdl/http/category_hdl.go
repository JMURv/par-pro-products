package http

import (
	"errors"
	metrics "github.com/JMURv/par-pro/products/internal/metrics/prometheus"
	repo "github.com/JMURv/par-pro/products/internal/repo"
	"github.com/JMURv/par-pro/products/internal/validation"
	"github.com/JMURv/par-pro/products/pkg/consts"
	"github.com/JMURv/par-pro/products/pkg/model"
	utils "github.com/JMURv/par-pro/products/pkg/utils/http"
	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

func RegisterCategoryRoutes(r *mux.Router, h *Handler) {
	r.HandleFunc("/api/category", h.listCategories).Methods(http.MethodGet)
	r.HandleFunc("/api/category", middlewareFunc(h.createCategory, h.authMiddleware)).Methods(http.MethodPost)
	r.HandleFunc("/api/category/search", h.categorySearch).Methods(http.MethodGet)
	r.HandleFunc("/api/category/filters/search", h.categoryFiltersSearch).Methods(http.MethodGet)
	r.HandleFunc("/api/category/{slug}", h.getCategory).Methods(http.MethodGet)
	r.HandleFunc("/api/category/{slug}", middlewareFunc(h.updateCategory, h.authMiddleware)).Methods(http.MethodPut)
	r.HandleFunc("/api/category/{slug}", middlewareFunc(h.deleteCategory, h.authMiddleware)).Methods(http.MethodDelete)
	r.HandleFunc("/api/category/{slug}/filters", h.listCategoryFilters).Methods(http.MethodGet)
}

func (h *Handler) categoryFiltersSearch(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.categoryFiltersSearch.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	query := r.URL.Query().Get("q")
	if len(query) < 3 {
		utils.SuccessResponse(w, c, []string{})
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = 10
	}

	res, err := h.ctrl.CategoryFiltersSearch(r.Context(), query, page, size)
	if err != nil {
		zap.L().Debug("failed to search filters", zap.String("op", op), zap.String("query", query), zap.Error(err))
		c = http.StatusInternalServerError
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessPaginatedResponse(w, c, res)
}

func (h *Handler) categorySearch(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.search.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	query := r.URL.Query().Get("q")
	if len(query) < 3 {
		utils.SuccessResponse(w, c, []string{})
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = 10
	}

	res, err := h.ctrl.CategorySearch(r.Context(), query, page, size)
	if err != nil {
		zap.L().Debug("failed to search categories", zap.String("op", op), zap.String("query", query), zap.Error(err))
		c = http.StatusInternalServerError
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessPaginatedResponse(w, c, res)
}

func (h *Handler) listCategoryFilters(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.listCategoryFilters.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	res, err := h.ctrl.ListCategoryFilters(r.Context(), mux.Vars(r)["slug"])
	if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to list category filters", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.listCategories.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = consts.DefaultPageSize
	}

	res, err := h.ctrl.ListCategories(r.Context(), page, size)
	if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to list categories", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessPaginatedResponse(w, c, res)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusCreated
	const op = "category.createCategory.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	req := &model.Category{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to decode request", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	if err := validation.CategoryValidation(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to validate obj", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	res, err := h.ctrl.CreateCategory(r.Context(), req)
	if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to create category", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.getCategory.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	res, err := h.ctrl.GetCategoryBySlug(r.Context(), mux.Vars(r)["slug"])
	if err != nil && errors.Is(err, repo.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("failed to found category", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to get category", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "category.updateCategory.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	req := &model.Category{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to decode request", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	if err := validation.CategoryValidation(req); err != nil {
		c = http.StatusBadRequest
		zap.L().Debug("failed to validate obj", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	res, err := h.ctrl.UpdateCategory(r.Context(), mux.Vars(r)["slug"], req)
	if err != nil && errors.Is(err, repo.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("category not found", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to update category", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessResponse(w, c, res)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusNoContent
	const op = "category.deleteCategory.handler"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	err := h.ctrl.DeleteCategory(r.Context(), mux.Vars(r)["slug"])
	if err != nil && errors.Is(err, repo.ErrNotFound) {
		c = http.StatusNotFound
		zap.L().Debug("category not found", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	} else if err != nil {
		c = http.StatusInternalServerError
		zap.L().Debug("failed to delete category", zap.String("op", op), zap.Error(err))
		utils.ErrResponse(w, c, err)
		return
	}

	utils.SuccessResponse(w, c, "OK")
}
