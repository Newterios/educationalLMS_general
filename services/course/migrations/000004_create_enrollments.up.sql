CREATE TABLE enrollments (
    id          UUID        PRIMARY KEY,
    course_id   UUID        NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    student_id  UUID        NOT NULL,
    enrolled_at TIMESTAMPTZ NOT NULL,
    UNIQUE (course_id, student_id)
);

CREATE INDEX idx_enrollments_student_id ON enrollments (student_id);
CREATE INDEX idx_enrollments_course_id  ON enrollments (course_id);
