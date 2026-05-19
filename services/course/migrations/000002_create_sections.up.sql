CREATE TABLE sections (
    id        UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title     TEXT NOT NULL,
    position  INT  NOT NULL DEFAULT 0
);

CREATE INDEX idx_sections_course_id ON sections (course_id);
