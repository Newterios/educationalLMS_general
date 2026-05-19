-- Runs once on a fresh Postgres volume (docker-entrypoint-initdb.d).
-- Creates one logical database per service as required by ARCHITECTURE.md §4.1.
-- On a shared VPS instance this keeps migrations fully isolated:
--   auth-svc  → auth_v2
--   course-svc → course_v2
--   assessment-svc → assessment_v2
CREATE DATABASE auth_v2;
CREATE DATABASE course_v2;
CREATE DATABASE assessment_v2;
