-- 1. Ускоряет поиск кандидатов в команде:
-- SELECT ... FROM users WHERE team_name = $1 AND is_active = TRUE
CREATE INDEX idx_users_team_active ON users(team_name, is_active);

-- 2. Ускоряет поиск PR конкретного автора:
-- SELECT ... FROM pull_requests WHERE author_id = $1
CREATE INDEX idx_pull_requests_author ON pull_requests(author_id);

-- 3. Ускоряет получение списка ревью (GetUsersGetReview):
-- SELECT ... JOIN pr_reviewers ... WHERE reviewer_id = $1
CREATE INDEX idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id);