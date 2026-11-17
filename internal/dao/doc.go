// Package dao is the data access layer of the service. Each DAO runs one PostgreSQL statement,
// embedded from the .sql file beside it.
//
// The schema keeps the bytes each project stores, in projects.stored_bytes. Triggers on beats, scenes, layers
// and cells (keep_stored_bytes, in migration 20261117201648_project_bytes_triggers) update the count at the end
// of every statement that writes them, a delete's cascade included, so no statement here maintains it.
package dao
