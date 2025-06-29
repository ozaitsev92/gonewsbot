.PHONY: migrate-status migrate-up migrate-down

migrate-status:
	goose status

migrate-up:
	goose up

migrate-down:
	goose down