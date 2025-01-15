create table if not exists users
(
    id                integer primary key,
    name              varchar   not null,
    created_at        timestamp not null default current_timestamp,
    discord_id        varchar not null,
    substack_session  varchar,
    substack_username varchar,
    kindle_mail       varchar
);

create table if not exists articles
(
    id           integer primary key,
    title        varchar   not null,
    author       varchar   not null,
    url          varchar   not null unique,
    release_date timestamp not null,
    local_path   varchar   not null,
    created_at   timestamp not null default current_timestamp,
    paid         boolean   not null default false,
    unique (title, author)
);

create table if not exists user_articles
(
    id         integer primary key,
    user_id    integer not null,
    article_id integer not null,
    created_at timestamp not null default current_timestamp,
    foreign key (user_id) references users (id),
    foreign key (article_id) references articles (id),
    unique (user_id, article_id)
);
