CREATE TABLE materials (
    id         UUID PRIMARY KEY,
    section_id UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    url        TEXT NOT NULL,
    title      TEXT NOT NULL
);

CREATE INDEX idx_materials_section_id ON materials (section_id);
