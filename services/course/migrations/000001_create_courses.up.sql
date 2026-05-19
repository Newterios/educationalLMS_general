CREATE TABLE courses (
    id          UUID        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    teacher_id  UUID        NOT NULL,
    language    TEXT        NOT NULL DEFAULT 'en',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_courses_teacher_id ON courses (teacher_id) WHERE deleted_at IS NULL;
