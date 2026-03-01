#!/bin/bash
set -e

export PGPASSWORD=$APP_DB_PASS;
# create app user and database
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  CREATE USER $APP_DB_USER WITH PASSWORD '$APP_DB_PASS';
  CREATE DATABASE $APP_DB_NAME;
  GRANT ALL PRIVILEGES ON DATABASE $APP_DB_NAME TO $APP_DB_USER;
  \c $APP_DB_NAME
  GRANT ALL ON SCHEMA public TO $APP_DB_USER;
EOSQL

psql --username "$APP_DB_USER" --dbname "$APP_DB_NAME" -f  /docker-entrypoint-initdb.d/02-create-tables.sql
psql --username "$APP_DB_USER" --dbname "$APP_DB_NAME" -f  /docker-entrypoint-initdb.d/03-data.sql

# create migration user and database
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  CREATE USER $MIGRATION_USER WITH PASSWORD '$MIGRATION_PASSWORD';
  CREATE DATABASE $MIGRATION_DATABASE;
  GRANT ALL PRIVILEGES ON DATABASE $MIGRATION_DATABASE TO $MIGRATION_USER;
  \c $MIGRATION_DATABASE
  GRANT ALL ON SCHEMA public TO $MIGRATION_USER;
EOSQL
