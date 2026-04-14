CREATE TABLE IF NOT EXISTS questions (
    id      UUID    PRIMARY KEY,
    quiz_id UUID    NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    body    TEXT    NOT NULL,
    choices JSONB   NOT NULL,
    points  INT     NOT NULL DEFAULT 1
);
