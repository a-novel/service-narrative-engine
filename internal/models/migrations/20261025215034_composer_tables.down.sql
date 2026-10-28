DROP INDEX IF EXISTS cells_project_id_layer_id_idx;

DROP TABLE IF EXISTS cells;

DROP INDEX IF EXISTS layers_project_id_manuscript_idx;

DROP TABLE IF EXISTS layers;

DROP TABLE IF EXISTS scenes;

DROP INDEX IF EXISTS projects_owner_id_idx;

DROP TABLE IF EXISTS projects;

DROP TYPE IF EXISTS project_language;
