.PHONY: migrate-status migrate-up migrate-down migrate-refresh

migrate-status:
	goose status

migrate-up:
	goose up

migrate-down:
	goose down

migrate-refresh: migrate-down migrate-up