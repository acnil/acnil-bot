# Hybrid Backend for AuditLambda

This document explains how to use the new hybrid backend approach where **only the audit functionality** moves to PostgreSQL while keeping all other data (games, members, juegatron) in Google Sheets.

## Overview

The auditlambda system now supports a hybrid approach:
- **Google Sheets** - Games, members, juegatron data (unchanged)
- **PostgreSQL** - Audit entries only (new option)

This provides the benefits of database storage for audit data while maintaining the existing Google Sheets workflow for game management.

## Architecture

### Data Storage Distribution:
- **Games** (`games` table in Google Sheets) - Game inventory, metadata, status
- **Members** (`Miembros Telegram` sheet) - User management, permissions
- **Juegatron** (`Juegos de mesa` sheet) - Juegatron-specific game tracking
- **Audit** (`audit_entries` table in PostgreSQL) - Change history, audit trail

### Lambda Functions:
- **bot_handler** - Uses Google Sheets for data, PostgreSQL for audit queries
- **audit_handler** - Reads from Google Sheets, writes to PostgreSQL

## PostgreSQL Setup

### 1. Database Schema

Run the simplified migration script (audit entries only):

```bash
psql -h localhost -U postgres -d acnil_audit -f migrations/001_audit_only_schema.sql
```

### 2. Environment Variables

To use the PostgreSQL audit backend, set these environment variables:

#### Required for PostgreSQL Audit:
```bash
# Audit backend selection
AUDIT_BACKEND=postgres

# Database connection (for audit only)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=acnil_audit
DB_SSLMODE=disable
```

#### Always Required (Google Sheets for games/members):
```bash
# Telegram bot
TOKEN=your_telegram_bot_token
WEBHOOK_SECRET_TOKEN=your_webhook_secret

# Google Sheets (games, members, juegatron)
SHEET_ID=your_sheet_id
AUDIT_SHEET_ID=your_audit_sheet_id  # Only used if AUDIT_BACKEND=sheets
JUEGATRON_SHEET_ID=your_juegatron_sheet_id
SHEETS_PRIVATE_KEY_ID=your_key_id
SHEETS_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
SHEETS_EMAIL=your_service_account@gserviceaccount.com
```

### 3. Deployment Examples

#### Using PostgreSQL for Audit:
```bash
export AUDIT_BACKEND=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=acnil_audit
export DB_SSLMODE=disable

# Google Sheets (still required)
export SHEET_ID=your_sheet_id
export JUEGATRON_SHEET_ID=your_juegatron_sheet_id
export TOKEN=your_telegram_bot_token

./lambda
```

#### Using Google Sheets for Audit (default):
```bash
export AUDIT_BACKEND=sheets
export SHEET_ID=your_sheet_id
export AUDIT_SHEET_ID=your_audit_sheet_id
export JUEGATRON_SHEET_ID=your_juegatron_sheet_id
export TOKEN=your_telegram_bot_token

./lambda
```

## Database Schema

### Audit Entries Table Only

```sql
CREATE TABLE audit_entries (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('new', 'removed', 'update')),
    game_id VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(100),
    holder VARCHAR(100),
    comments TEXT,
    take_date TIMESTAMP WITH TIME ZONE,
    return_date TIMESTAMP WITH TIME ZONE,
    price VARCHAR(50),
    publisher VARCHAR(100),
    bgg VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Indexes for Performance:
- `idx_audit_entries_timestamp` - For time-based queries
- `idx_audit_entries_game_id` - For game history
- `idx_audit_entries_type` - For change type filtering
- `idx_audit_entries_holder` - For user-specific queries

## Migration Strategy

### Step 1: Deploy with Google Sheets (Current)
```bash
AUDIT_BACKEND=sheets
# All other variables as usual
```

### Step 2: Set up PostgreSQL Database
```bash
# Create database
createdb acnil_audit

# Run migration
psql acnil_audit < migrations/001_audit_only_schema.sql
```

### Step 3: Switch to PostgreSQL Audit
```bash
AUDIT_BACKEND=postgres
# Add database connection variables
```

### Step 4: (Optional) Migrate Existing Audit Data
Export from Google Sheets "Audit" sheet and import into PostgreSQL:
```sql
INSERT INTO audit_entries (timestamp, type, game_id, name, ...)
VALUES (...);
```

## Terraform Configuration

### New Variables:
```hcl
variable "audit_backend" {
  description = "Backend to use for audit: sheets or postgres"
  type        = string
  default     = "sheets"
}

variable "db_host" {
  description = "PostgreSQL database host for audit"
  type        = string
  default     = "localhost"
}

variable "db_port" {
  description = "PostgreSQL database port for audit"
  type        = string
  default     = "5432"
}

variable "db_user" {
  description = "PostgreSQL database user for audit"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "PostgreSQL database password for audit"
  type        = string
  sensitive   = true
}

variable "db_name" {
  description = "PostgreSQL database name for audit"
  type        = string
  default     = "acnil_audit"
}

variable "db_sslmode" {
  description = "PostgreSQL SSL mode for audit"
  type        = string
  default     = "disable"
}
```

### Environment Variables:
Both Lambda functions now receive:
- `AUDIT_BACKEND` - Backend selection
- All Google Sheets variables (always required)
- All PostgreSQL variables (only used if `audit_backend = "postgres"`)

## Benefits of Hybrid Approach

### Advantages:
1. **Minimal Risk** - Only audit data moves to database
2. **Gradual Migration** - Can switch back to Google Sheets if needed
3. **Focused Scope** - Database only stores audit entries
4. **Simpler Schema** - No need to migrate complex game/member data
5. **Easier Testing** - Can test audit functionality independently
6. **Performance** - Faster audit queries with proper indexing
7. **Scalability** - Better handling of large audit histories

### Use Cases:
- **Large Audit Histories** - When audit sheet becomes slow
- **Advanced Analytics** - Complex audit reporting requirements
- **Data Retention** - Long-term audit storage needs
- **Compliance** - Audit trail requirements
- **Performance** - Faster audit history queries

## Troubleshooting

### Connection Issues:
- Verify PostgreSQL connection parameters
- Check if database exists: `psql -l`
- Test connection: `psql -h host -p port -U user -d db_name`

### Backend Switching:
- Can switch between `sheets` and `postgres` at runtime
- No data loss when switching backends
- Ensure both backends are accessible during transition

### Performance Issues:
- Check audit query performance with `EXPLAIN ANALYZE`
- Monitor database connection pool
- Consider archiving old audit entries

## Development

### Testing Both Backends:
```bash
# Test PostgreSQL audit
AUDIT_BACKEND=postgres go test ./...

# Test Google Sheets audit
AUDIT_BACKEND=sheets go test ./...
```

### Local Development:
```bash
# Use PostgreSQL for audit locally
export AUDIT_BACKEND=postgres
export DB_HOST=localhost
export DB_NAME=acnil_audit

# Keep Google Sheets for game data
export SHEET_ID=your_dev_sheet_id
```

## Architecture Benefits

The hybrid design maintains the excellent interface-based architecture:

```
Google Sheets Backend:
  GameDatabase <- SheetGameDatabase (games, members)
  JuegatronAuditDatabase <- SheetJuegatronAuditDatabase

PostgreSQL Backend:
  AuditDatabase <- PostgresAuditDatabase (audit only)

Bot Handler:
  GameDatabase (Google Sheets) + AuditDatabase (PostgreSQL/Sheets)

Audit Lambda:
  ROGameDatabase (Google Sheets) + AuditDatabase (PostgreSQL/Sheets)
```

This approach provides the best of both worlds: the familiar Google Sheets workflow for game management and the power of PostgreSQL for audit data.