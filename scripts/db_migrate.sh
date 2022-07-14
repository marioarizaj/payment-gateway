#!/bin/sh
# This script runs the DB migrations.
# migrate is a go binary, installed on the Docker image:
# https://github.con/gotang-aigrate/migrate/tree/master/cli
set -e

db_host=${DB_HOST:-"db"}
db_user=${DB_USER:-"db_user"}
db_pass=${DB_PASSWORD:-"db_pass"}
db_name=${DB_NAME:-"payments_gateway"}
db_env=${PG_ENV:-"dev"}

# Prevent engineers from accidentally doing something destructive.
#
if [ "$1" = "down" ] && [ -z "$2" ]; then
  echo "You're trying to run 'migrate down' without an integer argument."
  echo "That will rollback _all migrations, which is likely not what you want."
  echo " Please use down 1 to only rollback the last migration."
  echo ""
  exit 1;
fi

until pg_isready -h "${db_host}" -p 5432 -U "${db_user}"; do
  echo "$(date) - waiting for database to start"
  sleep 2
done

echo "Postgres is ready"

script_dir=$( cd "$(dirname "$0")"/; pwd);
migrations_path=$(cd "$script_dir/../migrations/"; pwd)


db_url="postgres://$db_user:$db_pass@$db_host:5432/$db_name?sslmode=disable"

cd "$migrations_path" && migrate --verbose --database "$db_url" --path . "$@"


if [ "$db_env" = "dev" ] || [ "$db_env" = "test" ]; then
  seeds_path=$(cd "$script_dir/../seeds/"; pwd)
  psql "$db_url" -f "$seeds_path"/seeds.sql
fi
