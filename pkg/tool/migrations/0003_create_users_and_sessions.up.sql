SET search_path TO curry_club;

create table users (
    id serial primary key,
    name text not null unique,
    credentials JSONB not null default '[]',
    created_at timestamp not null default now()
);

create table sessions (
    id serial primary key,
    session_data JSONB not null default '{}',
    created_at timestamp not null default now()
);
