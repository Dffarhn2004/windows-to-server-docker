module hcc.local/agent

go 1.23.5

require (
	github.com/shirou/gopsutil/v4 v4.25.8
	hcc.local/manifest v0.0.0
	hcc.local/observe v0.0.0
	hcc.local/redact v0.0.0
	hcc.local/sign v0.0.0
)

require (
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace hcc.local/manifest => ../pkg/manifest

replace hcc.local/observe => ../pkg/observe

replace hcc.local/redact => ../pkg/redact

replace hcc.local/sign => ../pkg/sign
