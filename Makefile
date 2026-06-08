.DEFAULT_GOAL := help
.PHONY: help test fmt lint typecheck clean

help:
	@echo "Available targets:"
	@echo "make test        - run tests"
	@echo "make fmt         - format code"
	@echo "make lint        - check code style"
	@echo "make typecheck   - run mypy"
	@echo "make clean       - remove cache files"

test:
	poetry run pytest

fmt:
	poetry run ruff format .
	poetry run ruff check --fix .

lint:
	poetry run ruff check .

typecheck:
	poetry run mypy jhansi/ tests/

clean:
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type d -name .pytest_cache -exec rm -rf {} +
	find . -type d -name .mypy_cache -exec rm -rf {} +
	find . -type d -name .ruff_cache -exec rm -rf {} +
