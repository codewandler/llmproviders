module github.com/codewandler/llmproviders

go 1.26.1

require (
	github.com/codewandler/agentapis v0.9.0
	github.com/codewandler/modeldb v0.0.0-00010101000000-000000000000
)

require github.com/google/uuid v1.6.0

replace github.com/codewandler/modeldb => ../modeldb

replace github.com/codewandler/agentapis => ../agentapis
