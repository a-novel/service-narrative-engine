-- Each scene takes back its beat's color, and its description and prompt where the scene holds none,
-- and the scenes renumber in narrative order. A project holding more than 500 scenes breaks the restored
-- position check, and the rollback fails.
ALTER TABLE scenes
DROP CONSTRAINT IF EXISTS scenes_beat_fkey,
DROP CONSTRAINT IF EXISTS scenes_beat_id_position_key,
DROP CONSTRAINT IF EXISTS scenes_position_check,
ADD COLUMN color text;

UPDATE scenes
SET
  color = beats.color,
  description = CASE
    WHEN scenes.description = '' THEN beats.description
    ELSE scenes.description
  END,
  prompt = CASE
    WHEN scenes.prompt = '' THEN beats.prompt
    ELSE scenes.prompt
  END,
  position = ordered.position
FROM
  beats,
  (
    SELECT
      scenes.id,
      (
        row_number() OVER (
          PARTITION BY
            scenes.project_id
          ORDER BY
            beats.position,
            scenes.position
        ) - 1
      )::integer AS position
    FROM
      scenes
      JOIN beats ON beats.id = scenes.beat_id
  ) AS ordered
WHERE
  beats.id = scenes.beat_id
  AND ordered.id = scenes.id;

ALTER TABLE scenes
ALTER COLUMN color
SET NOT NULL,
ADD CONSTRAINT scenes_color_check CHECK (color ~ '^#[0-9a-f]{6}$'),
DROP COLUMN IF EXISTS beat_id,
ADD CONSTRAINT scenes_position_check CHECK (
  position >= 0
  AND position < 500
),
ADD CONSTRAINT scenes_project_id_position_key UNIQUE (project_id, position)
DEFERRABLE INITIALLY DEFERRED;

DROP TABLE IF EXISTS beats;
