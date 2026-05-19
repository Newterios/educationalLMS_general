package routes

import (
	"encoding/json"
	"net/http"

	coursev1 "github.com/Newterios/lms-system-prob/proto/course/v1"
	"log/slog"
)

func RegisterCourse(mux *http.ServeMux, client coursev1.CourseServiceClient) {
	// Courses
	mux.HandleFunc("POST /api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		var req coursev1.CreateCourseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		resp, err := client.CreateCourse(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/courses/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.GetCourse(forwardAuth(r.Context(), r), &coursev1.GetCourseRequest{Id: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("PATCH /api/v1/courses/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req coursev1.UpdateCourseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.Id = r.PathValue("id")
		resp, err := client.UpdateCourse(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("DELETE /api/v1/courses/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.DeleteCourse(forwardAuth(r.Context(), r), &coursev1.DeleteCourseRequest{Id: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ListCourses(forwardAuth(r.Context(), r), &coursev1.ListCoursesRequest{})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	// Sections
	mux.HandleFunc("POST /api/v1/courses/{id}/sections", func(w http.ResponseWriter, r *http.Request) {
		var req coursev1.CreateSectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.CourseId = r.PathValue("id")
		resp, err := client.CreateSection(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/courses/{id}/sections", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ListSections(forwardAuth(r.Context(), r), &coursev1.ListSectionsRequest{CourseId: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	// Materials
	mux.HandleFunc("POST /api/v1/sections/{id}/materials", func(w http.ResponseWriter, r *http.Request) {
		var req coursev1.AddMaterialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid body")
			return
		}
		req.SectionId = r.PathValue("id")
		resp, err := client.AddMaterial(forwardAuth(r.Context(), r), &req)
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/sections/{id}/materials", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ListMaterials(forwardAuth(r.Context(), r), &coursev1.ListMaterialsRequest{SectionId: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	// Enrollments
	mux.HandleFunc("POST /api/v1/courses/{id}/enroll", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.EnrollStudent(forwardAuth(r.Context(), r), &coursev1.EnrollStudentRequest{CourseId: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("DELETE /api/v1/courses/{id}/enroll", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.UnenrollStudent(forwardAuth(r.Context(), r), &coursev1.UnenrollStudentRequest{CourseId: r.PathValue("id")})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	mux.HandleFunc("GET /api/v1/courses/{id}/enrollments", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.ListEnrollments(forwardAuth(r.Context(), r), &coursev1.ListEnrollmentsRequest{CourseId: stringPtr(r.PathValue("id"))})
		if err != nil {
			grpcError(w, err)
			return
		}
		jsonOK(w, resp)
	})

	slog.Info("course routes registered")
}

func stringPtr(s string) *string { return &s }
