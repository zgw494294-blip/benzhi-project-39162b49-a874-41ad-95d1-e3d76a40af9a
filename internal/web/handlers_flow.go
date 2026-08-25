package web

import (
	"net/http"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
)

func (h *Handler) PlanRevisions(w http.ResponseWriter, r *http.Request) {
	var command application.SubmitRevisionCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	command.PlanID = r.PathValue("planID")
	command.RequestKey = requestKey(r, command.RequestKey)
	h.executePlanCommand(w, r, func() (any, error) {
		return h.service.SubmitRevision(r.Context(), command)
	})
}

func (h *Handler) PlanChecks(w http.ResponseWriter, r *http.Request) {
	var command application.RunChecksCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	command.PlanID = r.PathValue("planID")
	command.RequestKey = requestKey(r, command.RequestKey)
	h.executePlanCommand(w, r, func() (any, error) {
		return h.service.RunChecks(r.Context(), command)
	})
}

func (h *Handler) PlanRehearsals(w http.ResponseWriter, r *http.Request) {
	var command application.RecordRehearsalCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	command.PlanID = r.PathValue("planID")
	command.RequestKey = requestKey(r, command.RequestKey)
	h.executePlanCommand(w, r, func() (any, error) {
		return h.service.RecordRehearsal(r.Context(), command)
	})
}

func (h *Handler) PlanReviews(w http.ResponseWriter, r *http.Request) {
	var command application.DecideReviewCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	command.PlanID = r.PathValue("planID")
	command.RequestKey = requestKey(r, command.RequestKey)
	h.executePlanCommand(w, r, func() (any, error) {
		return h.service.DecideReview(r.Context(), command)
	})
}

func (h *Handler) executePlanCommand(w http.ResponseWriter, _ *http.Request, execute func() (any, error)) {
	result, err := execute()
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AuthorizationVerification(w http.ResponseWriter, r *http.Request) {
	verification, err := h.service.VerifyAuthorization(r.Context(), r.PathValue("code"))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, verification)
}
