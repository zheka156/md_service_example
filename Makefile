local:
	@echo "Starting app locally"
	@heroku local:run go run main.go -e .env