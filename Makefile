build:
	go build -o=./tmp/bin/app .

watch:
	@go run github.com/air-verse/air@v1.61.7 \
		--build.cmd "make build" \
		--build.bin "./tmp/bin/app" \
		--build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tmpl, sql, html" \
		--misc.clean_on_exit "true"

templ:
	templ generate --watch --cmd="go run ." --proxy="http://localhost:9000"
