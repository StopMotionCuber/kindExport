create table users
(
    id                serial primary key,
    name              varchar   not null,
    created_at        timestamp not null default current_timestamp,
    discord_id        varchar,
    substack_session  varchar,
    substack_username varchar,
    kindle_mail       varchar
);

create table articles
(
    id         serial primary key,
    title      varchar   not null,
    author     varchar   not null,
    url        varchar   not null unique,
    local_path varchar   not null,
    created_at timestamp not null default current_timestamp,
    paid       boolean   not null default false,
    unique (title, author)
);
