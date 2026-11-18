-- A fixture runs after its matching migration. This one fails the roundtrip unless the recount left every
-- project's stored bytes equal to the sum of the bytes its rows hold.
DO $$
BEGIN
  IF EXISTS (
    SELECT
      1
    FROM
      projects
      LEFT JOIN (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          (
            SELECT project_id, bytes FROM beats
            UNION ALL
            SELECT project_id, bytes FROM scenes
            UNION ALL
            SELECT project_id, bytes FROM layers
            UNION ALL
            SELECT project_id, bytes FROM cells
          ) AS held
        GROUP BY
          project_id
      ) AS fresh ON fresh.project_id = projects.id
    WHERE
      projects.stored_bytes <> coalesce(fresh.bytes, 0)
  ) THEN
    RAISE EXCEPTION 'the recount left a stored bytes count apart from its rows';
  END IF;
END
$$;
