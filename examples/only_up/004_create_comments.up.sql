create table comments (
    id serial primary key,
    post_id integer not null references posts(id) on delete cascade,
    user_id integer not null references users(id) on delete cascade,
    content text not null,
    created_at timestamp default current_timestamp
);

create unique index idx_comments_post_user on comments(post_id, user_id, created_at);