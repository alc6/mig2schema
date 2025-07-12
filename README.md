# mig2schema

A tool that extracts database schema from PostgreSQL migration files with
MCP to discover the schema by running a temporary db.
Useful when you work on a project that has a migration folder containing a lot of files.

## Overview

`mig2schema` takes a directory containing PostgreSQL migration files (`.up.sql` and `.down.sql`) and extracts the resulting database schema after running the migrations. It uses testcontainers to spin up a temporary PostgreSQL instance, runs the migrations, and then extracts the schema information.

## Installation

```bash
go build -o mig2schema
```

## Usage

### Info Mode (Default)
Shows human-readable schema information:
```bash
./mig2schema /path/to/migrations
```

### Extract Mode
Outputs SQL CREATE statements that can be used to recreate the schema:
```bash
./mig2schema -e /path/to/migrations
./mig2schema --extract /path/to/migrations
```

## Migration File Format

The tool expects migration files to follow the naming convention:
- `001_create_users.up.sql` - Migration up file
- `001_create_users.down.sql` - Migration down file (optional)

Files are executed in alphabetical order by filename.

## Examples

### Info Mode Example
```bash
./mig2schema examples/migrations

# Output:
# === DATABASE SCHEMA ===
# Table: users
# Columns:
#   - id integer NOT NULL (PRIMARY KEY)
#   - email character varying NOT NULL
#   - username character varying NOT NULL
# Indexes:
#   - idx_users_email on (email)
#   - users_email_key on (email) (UNIQUE)
```

### Extract Mode Example
```bash
./mig2schema -e examples/migrations

# Output:
# create table users (
#     id integer not null default nextval('users_id_seq'::regclass),
#     email varchar(255) not null,
#     username varchar(255) not null,
#     primary key (id)
# );
# 
# create index idx_users_email on users (email);
# create unique index users_email_key on users (email);
```

## Requirements

- Docker (for testcontainers)
- Go 1.24.2+

## Using with Claude Code

This tool can be used as an MCP (Model Context Protocol) server in Claude Code:

```bash
claude mcp add mig2schema -- {path}/mig2schema --mcp
```

Where `{path}` is the full path to the mig2schema binary (e.g., `/Users/username/go/bin/mig2schema`).

Once added, Claude Code can use the following tools:
- `extract_schema`: Extract database schema from migration files
- `validate_migrations`: Validate migration files without running them