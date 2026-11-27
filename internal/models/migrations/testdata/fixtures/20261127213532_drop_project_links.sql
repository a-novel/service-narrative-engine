-- A fixture runs after its matching migration. The table went while the fixture project held a links row, so the
-- check fails the roundtrip unless the migration took those bytes out of the project's count.
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
          instances
        GROUP BY
          project_id
      ) AS fresh ON fresh.project_id = projects.id
    WHERE
      projects.stored_bytes <> coalesce(fresh.bytes, 0)
  ) THEN
    RAISE EXCEPTION 'a project counts stored bytes apart from its instances and their links';
  END IF;
END
$$;
