# Database migration staus
.PHONY: db-status
db-status:
	goose status

# Database migration
.PHONY: db-migrate
db-migrate:
	goose up

# Database rollback
.PHONY: db-down
db-down:
	goose down

# Database refresh
.PHONY: db-refresh
db-refresh: db-down db-migrate