-- A fixture runs after its matching migration. This one fails the roundtrip unless the migration reset the
-- stored bytes of the fixture project, whose rows it dropped.
DO $$
BEGIN
  IF EXISTS (
    SELECT
      1
    FROM
      projects
    WHERE
      stored_bytes <> 0
  ) THEN
    RAISE EXCEPTION 'the drop left a project counting the bytes of rows it no longer holds';
  END IF;
END
$$;
