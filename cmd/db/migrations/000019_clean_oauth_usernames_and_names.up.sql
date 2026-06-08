-- Strip the legacy "<letter><3 digits>-" prefix from Google usernames, but only
-- where the stripped handle is globally unique; colliding rows keep their prefix.
WITH stripped AS (
  SELECT u.id,
         regexp_replace(u.username, '^[A-Z][0-9]{3}-', '') AS new_username
  FROM users u
  JOIN oauth_accounts oa ON oa.user_id = u.id AND oa.provider = 'google'
  WHERE u.username ~ '^[A-Z][0-9]{3}-'
),
dupes AS (
  SELECT new_username FROM stripped GROUP BY new_username HAVING count(*) > 1
)
UPDATE users u
SET username = s.new_username, updated_at = NOW()
FROM stripped s
WHERE u.id = s.id
  AND s.new_username NOT IN (SELECT new_username FROM dupes)
  AND NOT EXISTS (
    SELECT 1 FROM users e WHERE e.username = s.new_username AND e.id <> s.id
  );

-- Drop the literal name placeholders; missing names should render blank.
UPDATE users u
SET first_name = '', updated_at = NOW()
FROM oauth_accounts oa
WHERE oa.user_id = u.id AND oa.provider = 'google' AND u.first_name = 'Google';

UPDATE users u
SET last_name = '', updated_at = NOW()
FROM oauth_accounts oa
WHERE oa.user_id = u.id AND oa.provider = 'google' AND u.last_name = 'User';
