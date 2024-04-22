-- SCIM users are remote users; their passwords are either synced from Okta or blank and they must
-- use SSO. Additionally, regular users can be marked as remote users.
UPDATE users SET remote = true WHERE id IN (SELECT user_id FROM scim.users);
