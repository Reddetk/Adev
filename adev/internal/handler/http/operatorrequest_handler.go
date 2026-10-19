package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type OperatorRequestHandler struct {
	usecase usecase.OperatorRequestUseCase
}

func NewOperatorRequestHandler(uc usecase.OperatorRequestUseCase) *OperatorRequestHandler {
	return &OperatorRequestHandler{usecase: uc}
}

// Create operatorrequest godoc
// @Summary Create operatorrequest
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param body body usecase.CreateOperatorRequestInput true "OperatorRequest payload"
// @Success 201 {object} usecase.CreateOperatorRequestOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /operatorrequests [post]
func (o *OperatorRequestHandler) CreateOperatorRequest(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateOperatorRequestInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	output, err := o.usecase.CreateOperatorRequest(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get operatorrequest by ID godoc
// @Summary Get operatorrequest by ID
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param id path int true "OperatorRequest ID"
// @Success 200 {object} domain.OperatorRequest
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /operatorrequests/{id} [get]
func (o *OperatorRequestHandler) GetOperatorRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid operatorrequest ID", http.StatusBadRequest)
		return
	}

	operatorrequest, err := o.usecase.GetOperatorRequest(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operatorrequest)
}

// Update operatorrequest godoc
// @Summary Update operatorrequest
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param id path int true "OperatorRequest ID"
// @Param body body usecase.UpdateOperatorRequestInput true "OperatorRequest payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /operatorrequests/{id} [put]
func (o *OperatorRequestHandler) UpdateOperatorRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid operatorrequest ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateOperatorRequestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := o.usecase.UpdateOperatorRequest(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete operatorrequest godoc
// @Summary Delete operatorrequest
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param id path int true "OperatorRequest ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /operatorrequests/{id} [delete]
func (o *OperatorRequestHandler) DeleteOperatorRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid operatorrequest ID", http.StatusBadRequest)
		return
	}

	if err := o.usecase.DeleteOperatorRequest(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List operatorrequests godoc
// @Summary List operatorrequests
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListOperatorRequestOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /operatorrequests [get]
func (o *OperatorRequestHandler) ListOperatorRequests(w http.ResponseWriter, r *http.Request) {
	output, err := o.usecase.ListOperatorRequests()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
