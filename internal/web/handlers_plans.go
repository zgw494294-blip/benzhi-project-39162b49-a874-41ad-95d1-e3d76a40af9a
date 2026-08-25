package web

import (
	"net/http"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
)

func (h *Handler) RiggingPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		query := application.PlanListQuery{
			State: r.URL.Query().Get("state"), Venue: r.URL.Query().Get("venue"),
			PerformanceDateFrom: r.URL.Query().Get("performanceDateFrom"),
			PerformanceDateTo:   r.URL.Query().Get("performanceDateTo"),
		}
		result, err := h.service.ListPlansFiltered(r.Context(), query)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	var command application.CreatePlanCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	command.RequestKey = requestKey(r, command.RequestKey)
	plan, err := h.service.CreatePlan(r.Context(), command)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/rigging-plans/"+plan.ID)
	writeJSON(w, http.StatusCreated, plan)
}

func (h *Handler) RiggingPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.service.GetPlan(r.Context(), r.PathValue("planID"))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}
