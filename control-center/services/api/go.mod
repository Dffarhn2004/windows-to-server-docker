module hcc.local/api

go 1.23.5

require (
	hcc.local/classify v0.0.0
	hcc.local/manifest v0.0.0
	hcc.local/observe v0.0.0
	hcc.local/redact v0.0.0
	hcc.local/sign v0.0.0
	modernc.org/sqlite v1.34.5
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.22.0 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
)

replace hcc.local/classify => ../../pkg/classify

replace hcc.local/manifest => ../../pkg/manifest

replace hcc.local/observe => ../../pkg/observe

replace hcc.local/redact => ../../pkg/redact

replace hcc.local/sign => ../../pkg/sign
