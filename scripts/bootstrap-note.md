# Bootstrap note

`playwright-go` downloads a platform driver zip from Playwright CDNs (`playwright.azureedge.net` mirrors).
As of 2026-07-19 those driver URLs return 404/400 from some networks; Chromium via `npx playwright` may still work.

Until the Go driver download works:

1. `make build` and `make test` succeed without the driver.
2. `make smoke` / live browser tools need `playwright install` for the Go driver to succeed.
3. Retry: `go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium`
4. Optional: set `PLAYWRIGHT_DOWNLOAD_HOST` if you mirror drivers yourself.
