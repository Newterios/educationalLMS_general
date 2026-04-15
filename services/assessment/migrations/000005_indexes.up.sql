-- Indexes on FK columns for fast JOIN / filter performance
CREATE INDEX IF NOT EXISTS idx_quizzes_course_id    ON quizzes  (course_id);
CREATE INDEX IF NOT EXISTS idx_quizzes_teacher_id   ON quizzes  (teacher_id);
CREATE INDEX IF NOT EXISTS idx_questions_quiz_id     ON questions(quiz_id);
CREATE INDEX IF NOT EXISTS idx_attempts_quiz_id      ON attempts (quiz_id);
CREATE INDEX IF NOT EXISTS idx_attempts_student_id   ON attempts (student_id);
