package antcache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDirectives(t *testing.T) {
	t.Run("has", func(t *testing.T) {
		var cases = []struct {
			title   string
			headers http.Header
			name    string
			has     bool
		}{
			{
				title:   "nil headers",
				headers: nil,
				name:    "no-cache",
				has:     false,
			},
			{
				title: "exists",
				headers: http.Header{
					"Cache-Control": []string{"no-cache"},
				},
				name: "no-cache",
				has:  true,
			},
			{
				title: "case insensitive",
				headers: http.Header{
					"Cache-Control": []string{"No-Cache"},
				},
				name: "no-cache",
				has:  true,
			},
			{
				title: "missing",
				headers: http.Header{
					"Cache-Control": []string{"max-age=5,must-revalidate"},
				},
				name: "no-cache",
				has:  false,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var d = directivesFrom(c.headers)

				assert.Equal(c.has, d.has(c.name))
			})
		}
	})

	t.Run("duration", func(t *testing.T) {
		var cases = []struct {
			title    string
			headers  http.Header
			name     string
			duration time.Duration
			ok       bool
		}{
			{
				title:    "nil headers",
				headers:  nil,
				name:     "max-age",
				duration: 0,
				ok:       false,
			},
			{
				title: "max-age",
				headers: http.Header{
					"Cache-Control": []string{"max-age=5"},
				},
				name:     "max-age",
				duration: 5 * time.Second,
				ok:       true,
			},
			{
				title: "negative max-age",
				headers: http.Header{
					"Cache-Control": []string{"max-age=-5"},
				},
				name:     "max-age",
				duration: -(5 * time.Second),
				ok:       true,
			},
			{
				title: "zero max-age",
				headers: http.Header{
					"Cache-Control": []string{"max-age=0"},
				},
				name:     "max-age",
				duration: 0,
				ok:       true,
			},
			{
				title: "invalid max-age",
				headers: http.Header{
					"Cache-Control": []string{"max-age=n"},
				},
				name:     "max-age",
				duration: 0,
				ok:       false,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var d = directivesFrom(c.headers)

				dur, ok := d.duration(c.name)

				assert.Equal(c.ok, ok)
				assert.Equal(c.duration, dur)
			})
		}
	})
}
