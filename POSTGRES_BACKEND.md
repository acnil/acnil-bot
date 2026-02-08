# PostgreSQL Backend for AuditLambda (Legacy Full Migration)

**⚠️ This document describes a full migration approach that has been superseded by the hybrid backend.**

**For the recommended approach, see [HYBRID_BACKEND.md](./HYBRID_BACKEND.md) which moves only audit data to PostgreSQL while keeping games and members in Google Sheets.**

This document explains how to use the PostgreSQL backend as a complete alternative to Google Sheets for the auditlambda system.

## Overview

The auditlambda system now supports two backends:
- **Google Sheets** (default) - Original implementation using Google Sheets
- **PostgreSQL** - New relational database backend

Both backends implement the same interfaces, so you can switch between them without changing the core audit logic.

## PostgreSQL Setup

### 1. Database Schema

Run the migration script to create the required tables:

```bash
psql -h localhost -U postgres -d acnil -f migrations/001_initial_schema.sql
```

### 2. Environment Variables

To use the PostgreSQL backend, set the following environment variables:

#### Required for PostgreSQL Backend:
```bash
# Backend selection
BACKEND=postgres

# Database connection
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=acnil
DB_SSLMODE=disable
```

#### Optional (for Google Sheets fallback):
```bash
# Only needed if BACKEND=sheets
SHEET_ID=your_sheet_id
AUDIT_SHEET_ID=your_audit_sheet_id
SHEETS_PRIVATE_KEY_ID=your_key_id
SHEETS_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
SHEETS_EMAIL=your_service_account@gserviceaccount.com
```

#### Always Required:
```bash
# Telegram bot
TOKEN=your_telegram_bot_token
```

### 3. Deployment Examples

#### Using PostgreSQL:
```bash
export BACKEND=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=acnil
export DB_SSLMODE=disable
export TOKEN=your_telegram_bot_token

./auditLambda
```

#### Using Google Sheets (default):
```bash
export BACKEND=sheets
export SHEET_ID=your_sheet_id
export AUDIT_SHEET_ID=your_audit_sheet_id
export TOKEN=your_telegram_bot_token

./auditLambda
```

## Database Schema

### Tables

#### `audit_entries`
Stores audit trail (replaces "Audit" sheet):
- `timestamp` - When the change occurred
- `type` - Type of change: 'new', 'removed', 'update'
- All game fields at the time of change

**Note:** Games and members data remain in Google Sheets. Only audit entries are stored in PostgreSQL.

### Indexes

Performance indexes are created for:
- Audit queries by timestamp and game_id
- Audit filtering by type and holder

## Migration from Google Sheets

To migrate from Google Sheets to PostgreSQL:

1. **Set up PostgreSQL database** with the schema
2. **Export data from Google Sheets**:
   - Games from "Juegos de mesa" sheet
   - Audit entries from "Audit" sheet  
   - Members from "Miembros Telegram" sheet
3. **Import data into PostgreSQL** using the appropriate INSERT statements
4. **Update environment variables** to use `BACKEND=postgres`
5. **Test the system** to ensure everything works correctly

## Performance Benefits

The PostgreSQL backend provides:
- **Faster queries** with proper indexing
- **Better scalability** for large datasets
- **ACID compliance** for data integrity
- **Concurrent access** support
- **Backup and recovery** options
- **Advanced analytics** capabilities

## Troubleshooting

### Connection Issues
- Verify database connection parameters
- Check if PostgreSQL is running
- Ensure firewall allows connections
- Validate SSL mode settings

### Permission Issues
- Ensure database user has proper permissions
- Check table ownership and privileges
- Verify connection string authentication

### Performance Issues
- Run `EXPLAIN ANALYZE` on slow queries
- Check if indexes are being used
- Monitor database connections
- Consider connection pooling

## Development

### Adding New Fields

When adding new fields to the Game or Member structs:
1. Update the PostgreSQL schema with ALTER TABLE
2. Update the INSERT/UPDATE queries in the database implementations
3. Handle backward compatibility for existing data

### Testing

Test both backends:
```bash
# Test PostgreSQL backend
BACKEND=postgres go test ./...

# Test Google Sheets backend  
BACKEND=sheets go test ./...
```

## Architecture

The PostgreSQL implementation follows the same interface-based design as the Google Sheets version:

```
AuditDatabase <- PostgresAuditDatabase
ROGameDatabase <- PostgresGameDatabase  
MembersDatabase <- PostgresMembersDatabase
```

This allows for easy switching between backends and future database implementations.