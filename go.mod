module github.com/codewandler/llmproviders

go 1.26.1

require (
	github.com/codewandler/agentapis v0.9.0
	github.com/codewandler/modeldb v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

replace github.com/codewandler/modeldb => ../modeldb

replace github.com/codewandler/agentapis => ../agentapis
