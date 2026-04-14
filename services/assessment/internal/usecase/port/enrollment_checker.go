package port

import (
	"context"

	"github.com/google/uuid"
)

// EnrollmentChecker verifies course enrollment by calling course-svc-v2.ListEnrollments
// with course_id + student_id filters (empty list ⇒ not enrolled).
// The gRPC client implementation lives in internal/client/course/.
// See ARCHITECTURE.md §3.4.
type EnrollmentChecker interface {
	IsEnrolled(ctx context.Context, courseID, studentID uuid.UUID) (bool, error)
}
