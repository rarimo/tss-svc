-- +migrate Up

create table sessions
(
    id          bigint primary key not null,
    status      integer            not null,
    indexes     text[] not null,
    root        text,
    proposer    text,
    signature   text,
    begin_block bigint             not null,
    end_block   bigint             not null,
    accepted    text[] not null,
    signed      text[] not null
);

-- +migrate Down
drop table sessions;