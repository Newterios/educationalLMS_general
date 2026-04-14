CREATE TABLE IF NOT EXISTS quizzes (
    id            UUID        PRIMARY KEY,
    course_id     UUID        NOT NULL,
    teacher_id    UUID        NOT NULL,
    title         TEXT        NOT NULL,
    time_limit_sec INT         NOT NULL DEFAULT 0,
    shuffle       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
