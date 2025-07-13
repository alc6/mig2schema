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

### Schema Extraction Providers

The tool supports multiple providers for extracting schema:

- **native** (default): Built-in provider using SQL queries to information_schema
- **pg_dump**: Uses PostgreSQL's pg_dump utility for complete DDL extraction

```bash
# List available providers
./mig2schema --list-providers

# Use native provider (default)
./mig2schema -p native -e /path/to/migrations

# Use pg_dump provider (requires pg_dump in PATH, only supports extract mode)
./mig2schema -p pg_dump -e /path/to/migrations
```

**Note**: The pg_dump provider only works with extract mode (`-e`) and provides more complete schema information including foreign keys, sequences, and all constraints.

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

Using native provider:
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

Using pg_dump provider (more complete output):
```bash
./mig2schema -p pg_dump -e examples/migrations

# Output includes:
# - Complete CREATE TABLE statements
# - ALTER TABLE statements for sequences
# - Foreign key constraints
# - All indexes with proper syntax
# - Default values and constraints
```

## Requirements

- Docker (for testcontainers)
- Go 1.24.2+
- PostgreSQL client tools (optional, required for pg_dump provider)

## Using with Claude Code

This tool can be used as an MCP (Model Context Protocol) server in Claude Code:

```bash
claude mcp add mig2schema -- {path}/mig2schema --mcp
```

Where `{path}` is the full path to the mig2schema binary (e.g., `/Users/username/go/bin/mig2schema`).

Once added, Claude Code can use the following tools:

### extract_schema
Extract database schema from migration files using pg_dump for complete DDL output.

Parameters:
- `migration_directory` (required): Path to directory containing migration files
- `format` (optional): Output format - "sql" (default and only option)

Example usage in Claude Code:
```
Use the extract_schema tool with migration_directory="./migrations"
```

**Note**: MCP mode always uses the pg_dump provider and outputs SQL DDL for the most complete and accurate schema extraction including foreign keys, sequences, and all constraints.

### validate_migrations
Validate migration files without running them.

Parameters:
- `migration_directory` (required): Path to directory containing migration files