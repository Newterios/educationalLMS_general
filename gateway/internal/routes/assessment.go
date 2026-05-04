package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"

	assessmentv1 "github.com/Newterios/lms-system-prob/proto/assessment/v1"
)

func RegisterAssessment(mux *http.ServeMux, client assessmentv1.AssessmentServiceClient) {
	// Quizzes
	mux.HandleFunc("POST /api/v1/assessments/quizzes", func(w http.ResponseWriter, r *http.Request) {
		var req assessmentv1.CreateQuizRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		resp, err := client.CreateQuiz(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/quizzes/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.GetQuiz(forwardAuth(r.Context(), r), &assessmentv1.GetQuizRequest{Id: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("PATCH /api/v1/assessments/quizzes/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req assessmentv1.UpdateQuizRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.Id = r.PathValue("id")
		resp, err := client.UpdateQuiz(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("DELETE /api/v1/assessments/quizzes/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.DeleteQuiz(forwardAuth(r.Context(), r), &assessmentv1.DeleteQuizRequest{Id: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/quizzes", func(w http.ResponseWriter, r *http.Request) {
		courseID := r.URL.Query().Get("course_id")
		resp, err := client.ListQuizzes(forwardAuth(r.Context(), r), &assessmentv1.ListQuizzesRequest{CourseId: courseID})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	// Attempts
	mux.HandleFunc("POST /api/v1/assessments/attempts", func(w http.ResponseWriter, r *http.Request) {
		var req assessmentv1.StartAttemptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		resp, err := client.StartAttempt(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/assessments/attempts/{id}/submit", func(w http.ResponseWriter, r *http.Request) {
		var req assessmentv1.SubmitAttemptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.AttemptId = r.PathValue("id")
		resp, err := client.SubmitAttempt(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/attempts/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.GetAttempt(forwardAuth(r.Context(), r), &assessmentv1.GetAttemptRequest{Id: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/attempts", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ListAttempts(forwardAuth(r.Context(), r), &assessmentv1.ListAttemptsRequest{
			QuizId:    optStr(r.URL.Query().Get("quiz_id")),
			StudentId: optStr(r.URL.Query().Get("student_id")),
		})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("POST /api/v1/assessments/attempts/{id}/grade", func(w http.ResponseWriter, r *http.Request) {
		var req assessmentv1.GradeSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.AttemptId = r.PathValue("id")
		resp, err := client.GradeSubmission(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/gradebook", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.GetGradebook(forwardAuth(r.Context(), r), &assessmentv1.GetGradebookRequest{
			CourseId: r.URL.Query().Get("course_id"),
		})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/assessments/gradebook/export", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ExportGrades(forwardAuth(r.Context(), r), &assessmentv1.ExportGradesRequest{
			CourseId: r.URL.Query().Get("course_id"),
		})
		if err != nil {
			grpcError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="`+resp.Filename+`"`)
		_, _ = w.Write(resp.Csv)
	})

	slog.Info("assessment routes registered")
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

