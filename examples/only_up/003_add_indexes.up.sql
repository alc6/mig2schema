create index idx_users_email on users(email);
create index idx_users_username on users(username);
create index idx_posts_user_id on posts(user_id);
create index idx_posts_published on posts(published);
create index idx_posts_created_at on posts(created_at);