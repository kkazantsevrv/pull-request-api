-- 1. Ускоряет поиск кандидатов в команде:
-- SELECT ... FROM users WHERE team_name = $1 AND is_active = TRUE
-- Composite Index (составной) работает эффективнее для фильтрации по двум полям сразу.
-- Также он покроет запросы просто по team_name (так как это префикс индекса).
CREATE INDEX idx_users_team_active ON users(team_name, is_active);

-- 2. Ускоряет поиск PR конкретного автора (частый кейс в UI):
-- SELECT ... FROM pull_requests WHERE author_id = $1
CREATE INDEX idx_pull_requests_author ON pull_requests(author_id);

-- 3. Ускоряет получение списка ревью (GetUsersGetReview):
-- SELECT ... JOIN pr_reviewers ... WHERE reviewer_id = $1
-- В таблице pr_reviewers Primary Key - это (pull_request_id, reviewer_id). 
-- Он автоматически создает индекс, но он эффективен только для поиска по pull_request_id.
-- Для поиска по reviewer_id нужен отдельный обратный индекс.
CREATE INDEX idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id);