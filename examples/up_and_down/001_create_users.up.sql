create table users (
    id serial primary key,
    email varchar(255) not null unique,
    username varchar(50) not null,
    password_hash varchar(255) not null,
    first_name varchar(100),
    last_name varchar(100),
    is_active boolean default true,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp
);