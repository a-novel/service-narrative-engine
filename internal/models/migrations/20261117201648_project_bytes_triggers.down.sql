-- The releases before these triggers keep the stored bytes in their own DAO statements.
DROP TRIGGER IF EXISTS cells_stored_bytes_delete ON cells;

DROP TRIGGER IF EXISTS cells_stored_bytes_update ON cells;

DROP TRIGGER IF EXISTS cells_stored_bytes_insert ON cells;

DROP TRIGGER IF EXISTS layers_stored_bytes_delete ON layers;

DROP TRIGGER IF EXISTS layers_stored_bytes_update ON layers;

DROP TRIGGER IF EXISTS layers_stored_bytes_insert ON layers;

DROP TRIGGER IF EXISTS scenes_stored_bytes_delete ON scenes;

DROP TRIGGER IF EXISTS scenes_stored_bytes_update ON scenes;

DROP TRIGGER IF EXISTS scenes_stored_bytes_insert ON scenes;

DROP TRIGGER IF EXISTS beats_stored_bytes_delete ON beats;

DROP TRIGGER IF EXISTS beats_stored_bytes_update ON beats;

DROP TRIGGER IF EXISTS beats_stored_bytes_insert ON beats;

DROP FUNCTION IF EXISTS keep_stored_bytes ();
