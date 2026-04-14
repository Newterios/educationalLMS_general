CREATE TABLE IF NOT EXISTS attempts (
    id           UUID        PRIMARY KEY,
    quiz_id      UUID        NOT NULL REFERENCES quizzes(id),
    student_id   UUID        NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    auto_score   DOUBLE PRECISION,
    manual_score DOUBLE PRECISION,
    status       TEXT        NOT NULL CHECK (status IN ('in_progress','submitted','graded')),
    answers      JSONB       NOT NULL DEFAULT '{}'
);
