#!/usr/bin/env bash
# smoke_assessment.sh — end-to-end smoke test for assessment-svc-v2
# Requires: grpcurl, jq, running auth-svc (50051), course-svc (50052), assessment-svc (50053)
# Usage: bash scripts/smoke_assessment.sh

set -euo pipefail

AUTH_ADDR="${AUTH_GRPC_TARGET:-localhost:50051}"
COURSE_ADDR="${COURSE_GRPC_TARGET:-localhost:50052}"
ASSESS_ADDR="${ASSESSMENT_GRPC_ADDR:-localhost:50053}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}▶${NC} $1"; }

# ── 0. Health check ───────────────────────────────────────────────────────────
info "Health check assessment-svc"
grpcurl -plaintext "$ASSESS_ADDR" grpc.health.v1.Health/Check \
  | jq -e '.status == "SERVING"' > /dev/null \
  && pass "assessment-svc healthy" || fail "assessment-svc unhealthy"

# ── 1. Login as teacher ───────────────────────────────────────────────────────
info "Teacher login"
TEACHER_TOKEN=$(grpcurl -plaintext -d '{"email":"teacher@test.com","password":"password123"}' \
  "$AUTH_ADDR" auth.v1.AuthService/Login 2>/dev/null | jq -r '.accessToken')
[ -n "$TEACHER_TOKEN" ] && [ "$TEACHER_TOKEN" != "null" ] \
  && pass "teacher login OK" || fail "teacher login failed"

# ── 2. Create a course ────────────────────────────────────────────────────────
info "Create course via course-svc"
COURSE_ID=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d '{"title":"Smoke Test Course","language":"en"}' \
  "$COURSE_ADDR" course.v1.CourseService/CreateCourse 2>/dev/null \
  | jq -r '.course.id')
[ -n "$COURSE_ID" ] && [ "$COURSE_ID" != "null" ] \
  && pass "course created: $COURSE_ID" || fail "create course failed"

# ── 3. Create quiz ────────────────────────────────────────────────────────────
info "CreateQuiz"
QUIZ_ID=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{
    \"course_id\": \"$COURSE_ID\",
    \"title\": \"Smoke Quiz\",
    \"time_limit_sec\": 60,
    \"shuffle\": false,
    \"questions\": [{
      \"body\": \"What is 2+2?\",
      \"points\": 1,
      \"choices\": [
        {\"key\":\"a\",\"value\":\"3\",\"correct\":false},
        {\"key\":\"b\",\"value\":\"4\",\"correct\":true}
      ]
    }]
  }" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/CreateQuiz 2>/dev/null \
  | jq -r '.quiz.id')
[ -n "$QUIZ_ID" ] && [ "$QUIZ_ID" != "null" ] \
  && pass "quiz created: $QUIZ_ID" || fail "CreateQuiz failed"

# ── 4. GetQuiz — verify correct field absent ──────────────────────────────────
info "GetQuiz — verify answer key NOT leaked"
CORRECT_FIELD=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"id\":\"$QUIZ_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/GetQuiz 2>/dev/null \
  | jq '.quiz.questions[0].choices[0].correct // empty')
[ -z "$CORRECT_FIELD" ] \
  && pass "correct field absent from GetQuiz response (no leakage)" \
  || fail "SECURITY: correct field leaked in GetQuiz response: $CORRECT_FIELD"

# ── 5. ListQuizzes ────────────────────────────────────────────────────────────
info "ListQuizzes"
COUNT=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"course_id\":\"$COURSE_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/ListQuizzes 2>/dev/null \
  | jq '.quizzes | length')
[ "$COUNT" -ge 1 ] && pass "ListQuizzes returned $COUNT quiz(zes)" || fail "ListQuizzes empty"

# ── 6. Login as student & enroll ──────────────────────────────────────────────
info "Student login"
STUDENT_TOKEN=$(grpcurl -plaintext -d '{"email":"student@test.com","password":"password123"}' \
  "$AUTH_ADDR" auth.v1.AuthService/Login 2>/dev/null | jq -r '.accessToken')
[ -n "$STUDENT_TOKEN" ] && [ "$STUDENT_TOKEN" != "null" ] \
  && pass "student login OK" || fail "student login failed"

info "Enroll student in course"
grpcurl -plaintext \
  -H "authorization: Bearer $STUDENT_TOKEN" \
  -d "{\"course_id\":\"$COURSE_ID\"}" \
  "$COURSE_ADDR" course.v1.CourseService/EnrollStudent > /dev/null 2>&1 \
  && pass "student enrolled" || fail "enroll failed"

# ── 7. StartAttempt ───────────────────────────────────────────────────────────
info "StartAttempt"
ATTEMPT_ID=$(grpcurl -plaintext \
  -H "authorization: Bearer $STUDENT_TOKEN" \
  -d "{\"quiz_id\":\"$QUIZ_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/StartAttempt 2>/dev/null \
  | jq -r '.attempt.id')
[ -n "$ATTEMPT_ID" ] && [ "$ATTEMPT_ID" != "null" ] \
  && pass "attempt started: $ATTEMPT_ID" || fail "StartAttempt failed"

# ── 8. Get question ID ────────────────────────────────────────────────────────
Q_ID=$(grpcurl -plaintext \
  -H "authorization: Bearer $STUDENT_TOKEN" \
  -d "{\"id\":\"$QUIZ_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/GetQuiz 2>/dev/null \
  | jq -r '.quiz.questions[0].id')

# ── 9. SubmitAttempt (correct answer: b=4) ────────────────────────────────────
info "SubmitAttempt with correct answer"
AUTO_SCORE=$(grpcurl -plaintext \
  -H "authorization: Bearer $STUDENT_TOKEN" \
  -d "{
    \"attempt_id\": \"$ATTEMPT_ID\",
    \"answers\": [{\"question_id\":\"$Q_ID\",\"choice_key\":\"b\"}]
  }" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/SubmitAttempt 2>/dev/null \
  | jq '.attempt.auto_score')
echo "  auto_score=$AUTO_SCORE"
[ "$(echo "$AUTO_SCORE >= 99" | bc -l)" -eq 1 ] \
  && pass "auto_score=100 (correct answer scored)" \
  || fail "expected auto_score=100, got $AUTO_SCORE"

# ── 10. GradeSubmission (teacher manual grade) ────────────────────────────────
info "GradeSubmission by teacher"
MANUAL_SCORE=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"attempt_id\":\"$ATTEMPT_ID\",\"manual_score\":95.0}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/GradeSubmission 2>/dev/null \
  | jq '.attempt.manual_score')
[ "$MANUAL_SCORE" = "95" ] || [ "$MANUAL_SCORE" = "95.0" ] \
  && pass "manual_score=$MANUAL_SCORE" || fail "GradeSubmission returned unexpected score: $MANUAL_SCORE"

# ── 11. GetGradebook ──────────────────────────────────────────────────────────
info "GetGradebook"
GB_COUNT=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"course_id\":\"$COURSE_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/GetGradebook 2>/dev/null \
  | jq '.entries | length')
[ "$GB_COUNT" -ge 1 ] && pass "gradebook has $GB_COUNT entry/entries" || fail "empty gradebook"

# ── 12. ExportGrades ──────────────────────────────────────────────────────────
info "ExportGrades"
FILENAME=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"course_id\":\"$COURSE_ID\"}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/ExportGrades 2>/dev/null \
  | jq -r '.filename')
[ -n "$FILENAME" ] && [ "$FILENAME" != "null" ] \
  && pass "ExportGrades filename: $FILENAME" || fail "ExportGrades returned no filename"

# ── 13. UpdateQuiz ────────────────────────────────────────────────────────────
info "UpdateQuiz"
NEW_TITLE=$(grpcurl -plaintext \
  -H "authorization: Bearer $TEACHER_TOKEN" \
  -d "{\"id\":\"$QUIZ_ID\",\"title\":\"Updated Quiz\",\"time_limit_sec\":90}" \
  "$ASSESS_ADDR" assessment.v1.AssessmentService/UpdateQuiz 2>/dev/null \
  | jq -r '.quiz.title')
[ "$NEW_TITLE" = "Updated Quiz" ] && pass "quiz title updated" || fail "UpdateQuiz failed: $NEW_TITLE"

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  assessment-svc-v2 smoke test PASSED   ${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
