//go:build generate
// +build generate

// Package mocks содержит сгенерированные моки для postgres репозитория.
package mocks

//go:generate mockgen -package=mocks -destination=dbpool_mock.go -source=../pool.go DBPool
