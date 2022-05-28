module github.com/openucx/openhpca/tools

go 1.14

require (
	github.com/gvallee/go_benchmark v1.0.0
	github.com/gvallee/go_hpc_jobmgr v1.3.1
	github.com/gvallee/go_osu v1.7.1
	github.com/gvallee/go_software_build v1.1.2
	github.com/gvallee/go_util v1.5.1
	github.com/gvallee/go_workspace v1.2.1
	github.com/gvallee/validation_tool v1.3.1
	gonum.org/v1/plot v0.9.0
)

replace github.com/gvallee/go_hpc_jobmgr v1.3.1 => github.com/BrodyWilliams/go_hpc_jobmgr v1.3.2-0.20220528001707-b0cfa6925f60
