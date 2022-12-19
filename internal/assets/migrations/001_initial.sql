-- +migrate Up

create table sessions
(
    id           bigint primary key not null,
    status       integer            not null,
    begin_block  bigint             not null,
    end_block    bigint             not null,
    session_type integer,
    data_id      bigint unique
);

create table default_session_data
(
    id        bigint primary key not null,
    parties   text[] not null,
    proposer  text,
    indexes   text[] not null,
    root      text,
    accepted  text[] not null,
    signature text
);

create table reshare_session_data
(
    id            bigint primary key not null,
    parties       text[] not null,
    proposer      text,
    old_key       text,
    new_key       text,
    key_signature text,
    signature     text,
    root          text
);

create table keygen_session_data
(
    id      bigint primary key not null,
    parties text[] not null,
    key     text
);


-- +migrate Down
drop table sessions;
drop table default_session_data;
drop table reshare_session_data;
drop table keygen_session_data;