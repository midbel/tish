package tish

import (
	"strings"
)

func Match(str, pattern string) bool {
	if pattern == "*" || str == pattern {
		return true
	}
	var i, j int
	for i < len(pattern) && j < len(str) {
		switch pattern[i] {
		case '*':
			i++
			ni, nj, ok := starMatch(str[j:], pattern[i:])
			if !ok {
				return false
			}
			i += ni
			j += nj
		case '[':
			i++
			ni, ok := setMatch(str[j], pattern[i:])
			if !ok {
				return false
			}
			i += ni
			j++
		case '?':
			i++
			j++
		default:
			if pattern[i] != str[j] {
				return false
			}
			i++
			j++
		}
	}
	return i == len(pattern) && j == len(str)
}

func setMatch(char byte, pattern string) (int, bool) {
	var (
		i   int
		ok  bool
		neg bool
	)
	if pattern[i] == '!' || pattern[i] == '^' {
		neg = true
		i++
	}
	for i < len(pattern) && pattern[i] != ']' {
		if pattern[i] == '-' {
			prev, next := pattern[i-1], pattern[i+1]
			if isRange(prev, next) {
				if ok = prev <= char && char <= next; ok {
					break
				}
			}
		}
		if ok = char == pattern[i]; ok {
			break
		}
		i++
	}
	for pattern[i] != ']' && i < len(pattern) {
		i++
	}
	if neg {
		ok = !ok
	}
	return i + 1, ok
}

func starMatch(str, pattern string) (int, int, bool) {
	var i, j int

	for i < len(pattern) {
		if pattern[i] != '*' {
			break
		}
	}
	if i >= len(pattern) {
		return i, len(str), true
	}

	for j < len(str) {
		if Match(str[j:], pattern) {
			return i, j, true
		}
		j++
	}
	return 0, 0, false
}

func isRange(prev, next byte) bool {
	return prev < next && acceptRange(prev) && acceptRange(next)
}

func acceptRange(b byte) bool {
	return (b >= 'a' || b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func hasMeta(str string) bool {
	return strings.ContainsAny(str, "*?[")
}
