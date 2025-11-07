package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

type Clause struct {
	Field    string
	Operator string
	Value    string
}

type Query struct {
	SelectFields []string
	Filters      []Clause
	Limit        int
}

var (
	selectRegex = regexp.MustCompile(`select\s+([a-zA-Z0-9_,\s]+)`)
	whereRegex  = regexp.MustCompile(`where\s+(.+?)(?:\s+limit|$)`)
	limitRegex  = regexp.MustCompile(`limit\s+(\d+)`)
	filterRegex = regexp.MustCompile(`([a-zA-Z0-9_]+)\s*(=|!=|>|<|>=|<=|in)\s*([^,]+)`)
)

func Parse(input string) (Query, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if !strings.HasPrefix(input, "select") {
		return Query{}, fmt.Errorf("query must start with select")
	}

	var query Query

	selectMatch := selectRegex.FindStringSubmatch(input)
	if len(selectMatch) < 2 {
		return Query{}, fmt.Errorf("missing select fields")
	}
	fields := strings.Split(selectMatch[1], ",")
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		query.SelectFields = append(query.SelectFields, field)
	}

	if whereMatch := whereRegex.FindStringSubmatch(input); len(whereMatch) >= 2 {
		filters := filterRegex.FindAllStringSubmatch(whereMatch[1], -1)
		for _, match := range filters {
			if len(match) < 4 {
				continue
			}
			query.Filters = append(query.Filters, Clause{
				Field:    strings.TrimSpace(match[1]),
				Operator: match[2],
				Value:    strings.TrimSpace(match[3]),
			})
		}
	}

	if limitMatch := limitRegex.FindStringSubmatch(input); len(limitMatch) >= 2 {
		fmt.Sscanf(limitMatch[1], "%d", &query.Limit)
	}

	if len(query.SelectFields) == 0 {
		return Query{}, fmt.Errorf("at least one field must be selected")
	}

	return query, nil
}
