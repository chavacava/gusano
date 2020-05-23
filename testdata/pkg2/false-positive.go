package pkg2

import (
	"regexp"
	"strconv"
)

// Positions of elements in a log entry
const ipPos = 1
const identityPos = 2
const personIDPos = 3
const datePos = 4
const reqMethodPos = 5
const reqResourcePos = 6
const reqProtocolPos = 7
const statusPos = 8
const sizePos = 9
const totalElements = 9

// if you have a problem and you use regexp to resolve it, now you have two problems
const re = `^(\S+) (\S+) (\S+) \[([\w:/]+\s[+\-]\d{4})\] "(\S+)\s?(\S+)?\s?(\S+)?" (\d{3}|-) (\d+|-)\s?$`

var clfRegExp = regexp.MustCompile(re)

// EntryFromLogLine yields an Entry from the given log line
func EntryFromLogLine(line string) (string, error) {
	const unknown = 0
	elements := clfRegExp.FindStringSubmatch(line)

	status, err := strconv.Atoi(elements[statusPos])
	if err != nil {
		status = unknown
	}

	_ = status
	size, err := strconv.Atoi(elements[sizePos])
	if err != nil {
		size = unknown
	}
	_ = size

	_ = elements[ipPos]
	_ = elements[identityPos]
	_ = elements[personIDPos]
	_ = elements[datePos]
	_ = elements[reqMethodPos]
	_ = elements[reqResourcePos]
	_ = elements[reqProtocolPos]
	_ = status
	_ = size

	return "", nil
}
