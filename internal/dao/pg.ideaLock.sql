SELECT
  id
FROM
  ideas
WHERE
  id = ?0
  AND owner_id = ?1
FOR UPDATE;
