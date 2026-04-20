module github.com/codewandler/llmproviders

go 1.26.1

require (
	github.com/codewandler/agentapis v0.7.1
	github.com/codewandler/modeldb v0.0.0-00010101000000-000000000000
)

replace github.com/codewandler/modeldb => ../modeldb

replace github.com/codewandler/agentapis => ../agentapis
