SET search_path TO curry_club;

create table users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text not null unique,
    credentials JSONB not null default '[]',
    created_at timestamp not null default now()
);

create table sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_data JSONB not null default '{}',
    created_at timestamp not null default now()
);
