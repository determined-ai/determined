#!/bin/bash

# make -C tools start-db

psql -h localhost -U postgres -d postgres <<EOF
  drop database determined;
  create database determined;
  \c determined;
  \i ../backup/database/21-05-11_16-16-44.sql;

  /* pg wouldn't find the tables without doing this for some reason */
  \c postgres;
  \c determined;

  /* this will fail without the migrations */
  select id, notes from experiments;
  /* old experiments won't have name */
  select id, config->>'name' as name, config->>'description' as description from experiments;

  /* run the migrations */
  \i ./master/static/migrations/20210512161723_add-experiment-notes.up.sql;
  \i ./master/static/migrations/20210511152412_description-to-name.up.sql;

  select id, notes from experiments;
  select id, config->>'name' as name, config->>'description' as description from experiments;

EOF
