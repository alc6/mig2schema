create table posts (
    id serial primary key,
    title varchar(255) not null,
    content text,
    user_id integer not null references users(id) on delete cascade,
    published boolean default false,
    published_at timestamp,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);