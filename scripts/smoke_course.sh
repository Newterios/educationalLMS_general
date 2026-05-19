#!/usr/bin/env bash
# Smoke test for course-svc-v2 (Phase 1E).
# Prerequisites: grpcurl, docker, go 1.25+
# Usage:
#   make smoke-course
#   # or directly:
#   DB_URL=postgres://... JWT_ACCESS_SECRET=... JWT_REFRESH_SECRET=... bash scripts/smoke_course.sh

set -euo pipefail

AUTH_HOST="127.0.0.1:50051"
COURSE_HOST="127.0.0.1:50052"
COMPOSE="docker compose -p edulmsv2 -f docker-compose.dev.yml"

# ── env defaults for local dev ─────────────────────────────────────────────
export DB_URL="${DB_URL:-postgres://edulms:edulms@127.0.0.1:5433/auth_v2?sslmode=disable}"
export DATABASE_URL_COURSE="${DATABASE_URL_COURSE:-postgres://edulms:edulms@127.0.0.1:5433/course_v2?sslmode=disable}"
export JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-smoke-access-secret-dev}"
export JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-smoke-refresh-secret-dev}"
export GRPC_PORT="${GRPC_PORT:-50051}"
export COURSE_GRPC_PORT="${COURSE_GRPC_PORT:-50052}"
export MIGRATIONS_DIR="${MIGRATIONS_DIR:-services/auth/migrations}"
export COURSE_MIGRATIONS_DIR="${COURSE_MIGRATIONS_DIR:-services/course/migrations}"

# ── guards ────────────────────────────────────────────────────────────────
for cmd in grpcurl docker go; do
  command -v "$cmd" >/dev/null || { echo "ERROR: $cmd not found in PATH"; exit 1; }
done

TEACHER_EMAIL="teacher.$(date +%s)@example.com"
STUDENT_EMAIL="student.$(date +%s)@example.com"

AUTH_PID=""
COURSE_PID=""

# Pre-flight: release the ports if leftover processes are holding them
lsof -ti :"$GRPC_PORT" 2>/dev/null | xargs kill -9 2>/dev/null || true
lsof -ti :"$COURSE_GRPC_PORT" 2>/dev/null | xargs kill -9 2>/dev/null || true

cleanup() {
  echo ""
  echo "── cleanup ─────────────────────────────────────────────────────────────"
  if [[ -n "$AUTH_PID" ]] && kill -0 "$AUTH_PID" 2>/dev/null; then
    kill "$AUTH_PID" 2>/dev/null; wait "$AUTH_PID" 2>/dev/null || true
    echo "auth-svc stopped (pid $AUTH_PID)"
  fi
  if [[ -n "$COURSE_PID" ]] && kill -0 "$COURSE_PID" 2>/dev/null; then
    kill "$COURSE_PID" 2>/dev/null; wait "$COURSE_PID" 2>/dev/null || true
    echo "course-svc stopped (pid $COURSE_PID)"
  fi
  $COMPOSE stop postgres 2>/dev/null || true
  echo "done."
}
trap cleanup EXIT

# ── 1. infra ──────────────────────────────────────────────────────────────
$COMPOSE down -v >/dev/null 2>&1 || true
echo "── starting postgres ────────────────────────────────────────────────────"
$COMPOSE up -d postgres
echo "waiting for postgres..."
for i in $(seq 1 30); do
  $COMPOSE exec -T postgres pg_isready -U edulms >/dev/null 2>&1 && break
  sleep 1
done

# ── 2. auth binary ────────────────────────────────────────────────────────
echo "── starting auth-svc-v2 ─────────────────────────────────────────────────"
go run ./services/auth/cmd/auth &
AUTH_PID=$!

echo "waiting for auth health (pid $AUTH_PID)..."
for i in $(seq 1 30); do
  grpcurl -plaintext "$AUTH_HOST" grpc.health.v1.Health/Check >/dev/null 2>&1 && break
  kill -0 "$AUTH_PID" 2>/dev/null || { echo "ERROR: auth-svc exited"; exit 1; }
  sleep 1
done
grpcurl -plaintext "$AUTH_HOST" grpc.health.v1.Health/Check | grep -q "SERVING" \
  || { echo "ERROR: auth health not SERVING"; exit 1; }
echo "auth health OK"

# ── 3. course binary ──────────────────────────────────────────────────────
echo "── starting course-svc-v2 ───────────────────────────────────────────────"
go run ./services/course/cmd/course &
COURSE_PID=$!

echo "waiting for course health (pid $COURSE_PID)..."
for i in $(seq 1 30); do
  grpcurl -plaintext "$COURSE_HOST" grpc.health.v1.Health/Check >/dev/null 2>&1 && break
  kill -0 "$COURSE_PID" 2>/dev/null || { echo "ERROR: course-svc exited"; exit 1; }
  sleep 1
done
grpcurl -plaintext "$COURSE_HOST" grpc.health.v1.Health/Check | grep -q "SERVING" \
  || { echo "ERROR: course health not SERVING"; exit 1; }
echo "course health OK"

# ── helpers ───────────────────────────────────────────────────────────────
auth_rpc() {
  local method="$1"; shift
  grpcurl -plaintext "$@" "$AUTH_HOST" "auth.v1.AuthService/$method"
}
course_rpc() {
  local method="$1"; shift
  grpcurl -plaintext "$@" "$COURSE_HOST" "course.v1.CourseService/$method"
}
jval() { echo "$1" | grep -o "\"$2\": *\"[^\"]*\"" | head -1 | sed 's/.*": *"\(.*\)"/\1/'; }
jnum() { echo "$1" | grep -o "\"$2\": *[0-9]*" | head -1 | sed 's/.*": *\([0-9]*\)/\1/'; }
pass() { echo "  ✓ $1"; }
fail() { echo "  ✗ $1"; exit 1; }

echo ""
echo "── smoke sequence ───────────────────────────────────────────────────────"

# ── A. Register & login teacher ──────────────────────────────────────────────
echo "A. Register teacher"
auth_rpc Register -d "{\"email\":\"$TEACHER_EMAIL\",\"password\":\"teacher1234\",\"full_name\":\"Teacher\",\"locale\":\"en\"}" >/dev/null
pass "Teacher registered"

echo "B. Login teacher"
TEACHER_LOGIN=$(auth_rpc Login -d "{\"email\":\"$TEACHER_EMAIL\",\"password\":\"teacher1234\"}")
TEACHER_TOKEN=$(jval "$TEACHER_LOGIN" "accessToken")
[[ -n "$TEACHER_TOKEN" ]] || fail "Teacher login: no accessToken"
TEACHER_AUTH="authorization: Bearer $TEACHER_TOKEN"
pass "Teacher login OK"

# ── B. Register & login student ──────────────────────────────────────────────
echo "C. Register student"
STUDENT_REG=$(auth_rpc Register -d "{\"email\":\"$STUDENT_EMAIL\",\"password\":\"student1234\",\"full_name\":\"Student\",\"locale\":\"en\"}")
STUDENT_ID=$(jval "$STUDENT_REG" "userId")
[[ -n "$STUDENT_ID" ]] || fail "Student register: no userId"
pass "Student registered (id=$STUDENT_ID)"

echo "D. Login student"
STUDENT_LOGIN=$(auth_rpc Login -d "{\"email\":\"$STUDENT_EMAIL\",\"password\":\"student1234\"}")
STUDENT_TOKEN=$(jval "$STUDENT_LOGIN" "accessToken")
[[ -n "$STUDENT_TOKEN" ]] || fail "Student login: no accessToken"
STUDENT_AUTH="authorization: Bearer $STUDENT_TOKEN"
pass "Student login OK"

# ── C. CreateCourse ──────────────────────────────────────────────────────────
echo "1. CreateCourse"
CREATE_RESP=$(course_rpc CreateCourse -H "$TEACHER_AUTH" \
  -d '{"title":"Go Fundamentals","description":"Learn Go","language":"en"}')
COURSE_ID=$(jval "$CREATE_RESP" "id")
[[ -n "$COURSE_ID" ]] || fail "CreateCourse: no course id"
pass "CreateCourse → id=$COURSE_ID"

# ── D. GetCourse ──────────────────────────────────────────────────────────────
echo "2. GetCourse"
GET_RESP=$(course_rpc GetCourse -H "$TEACHER_AUTH" -d "{\"id\":\"$COURSE_ID\"}")
echo "$GET_RESP" | grep -q "Go Fundamentals" || fail "GetCourse: title not found"
pass "GetCourse returned correct title"

# ── E. UpdateCourse ───────────────────────────────────────────────────────────
echo "3. UpdateCourse"
UPD_RESP=$(course_rpc UpdateCourse -H "$TEACHER_AUTH" \
  -d "{\"id\":\"$COURSE_ID\",\"title\":\"Go Advanced\"}")
echo "$UPD_RESP" | grep -q "Go Advanced" || fail "UpdateCourse: title not updated"
pass "UpdateCourse OK"

# ── F. CreateSection ─────────────────────────────────────────────────────────
echo "4. CreateSection"
SECTION_RESP=$(course_rpc CreateSection -H "$TEACHER_AUTH" \
  -d "{\"course_id\":\"$COURSE_ID\",\"title\":\"Chapter 1\",\"position\":1}")
SECTION_ID=$(jval "$SECTION_RESP" "id")
[[ -n "$SECTION_ID" ]] || fail "CreateSection: no section id"
pass "CreateSection → id=$SECTION_ID"

# ── G. AddMaterial ────────────────────────────────────────────────────────────
echo "5. AddMaterial"
MAT_RESP=$(course_rpc AddMaterial -H "$TEACHER_AUTH" \
  -d "{\"section_id\":\"$SECTION_ID\",\"kind\":\"link\",\"url\":\"https://go.dev\",\"title\":\"Go Homepage\"}")
MAT_ID=$(jval "$MAT_RESP" "id")
[[ -n "$MAT_ID" ]] || fail "AddMaterial: no material id"
pass "AddMaterial → id=$MAT_ID"

# ── H. ListSections ──────────────────────────────────────────────────────────
echo "6. ListSections"
SECS_RESP=$(course_rpc ListSections -H "$TEACHER_AUTH" -d "{\"course_id\":\"$COURSE_ID\"}")
echo "$SECS_RESP" | grep -q "Chapter 1" || fail "ListSections: section not found"
pass "ListSections OK"

# ── I. ListMaterials ──────────────────────────────────────────────────────────
echo "7. ListMaterials"
MATS_RESP=$(course_rpc ListMaterials -H "$TEACHER_AUTH" -d "{\"section_id\":\"$SECTION_ID\"}")
echo "$MATS_RESP" | grep -q "Go Homepage" || fail "ListMaterials: material not found"
pass "ListMaterials OK"

# ── J. EnrollStudent (self-enroll) ────────────────────────────────────────────
echo "8. EnrollStudent (self-enroll)"
ENROLL_RESP=$(course_rpc EnrollStudent -H "$STUDENT_AUTH" \
  -d "{\"course_id\":\"$COURSE_ID\",\"student_id\":\"$STUDENT_ID\"}")
ENROLL_ID=$(jval "$ENROLL_RESP" "id")
[[ -n "$ENROLL_ID" ]] || fail "EnrollStudent: no enrollment id"
pass "EnrollStudent → id=$ENROLL_ID"

# ── K. ListEnrollments ────────────────────────────────────────────────────────
echo "9. ListEnrollments"
ENROLL_LIST=$(course_rpc ListEnrollments -H "$TEACHER_AUTH" \
  -d "{\"course_id\":\"$COURSE_ID\"}")
echo "$ENROLL_LIST" | grep -q "$STUDENT_ID" || fail "ListEnrollments: student not found"
pass "ListEnrollments returned enrolled student"

# ── L. Negative: student cannot update course ─────────────────────────────────
echo "10. Negative: student UpdateCourse must fail with PermissionDenied"
UPD_FAIL=$(course_rpc UpdateCourse -H "$STUDENT_AUTH" \
  -d "{\"id\":\"$COURSE_ID\",\"title\":\"Hacked\"}" 2>&1 || true)
echo "$UPD_FAIL" | grep -qi "PermissionDenied\|permission_denied" \
  || fail "UpdateCourse by student: expected PermissionDenied"
pass "Student cannot update teacher's course"

# ── M. UnenrollStudent ────────────────────────────────────────────────────────
echo "11. UnenrollStudent (self-unenroll)"
course_rpc UnenrollStudent -H "$STUDENT_AUTH" \
  -d "{\"course_id\":\"$COURSE_ID\",\"student_id\":\"$STUDENT_ID\"}" >/dev/null
pass "UnenrollStudent OK"

# ── N. ListCourses ────────────────────────────────────────────────────────────
echo "12. ListCourses"
LIST_RESP=$(course_rpc ListCourses -H "$TEACHER_AUTH" -d '{}')
echo "$LIST_RESP" | grep -q "$COURSE_ID" || fail "ListCourses: course not found"
pass "ListCourses returned created course"

# ── O. DeleteCourse ───────────────────────────────────────────────────────────
echo "13. DeleteCourse"
course_rpc DeleteCourse -H "$TEACHER_AUTH" -d "{\"id\":\"$COURSE_ID\"}" >/dev/null
pass "DeleteCourse OK"

# ── P. GetCourse after delete must fail ──────────────────────────────────────
echo "14. GetCourse on deleted course must fail with NotFound"
GET_DEL=$(course_rpc GetCourse -H "$TEACHER_AUTH" -d "{\"id\":\"$COURSE_ID\"}" 2>&1 || true)
echo "$GET_DEL" | grep -qi "NotFound\|not_found" || fail "GetCourse after delete: expected NotFound"
pass "GetCourse returns NotFound for deleted course"

echo ""
echo "══════════════════════════════════════════════════════════════════════"
echo "  ALL COURSE SMOKE CHECKS PASSED"
echo "══════════════════════════════════════════════════════════════════════"
