package router

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const hostPattern = "^.*ip([0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3})-([0-9|a-z]+)(?:-?([0-9]{1,5}))?(?:\\.([a-z|A-Z|0-9|_|\\-\\.]+))?(?:\\:([0-9]{1,5}))?$"

var hostRegex *regexp.Regexp

func init() {
	hostRegex = regexp.MustCompile(hostPattern)
}

type HostOpts struct {
	TLD         string
	EncodedPort int
	Port        int
}

type HostInfo struct {
	SessionId   string
	InstanceIP  string
	TLD         string
	EncodedPort int
	Port        int
}

func EncodeHost(sessionId, instanceIP string, opts HostOpts) string {
	encodedIP := strings.Replace(instanceIP, ".", "-", -1)

	sub := fmt.Sprintf("ip%s-%s", encodedIP, sessionId)
	if opts.EncodedPort > 0 {
		sub = fmt.Sprintf("%s-%d", sub, opts.EncodedPort)
	}
	if opts.TLD != "" {
		sub = fmt.Sprintf("%s.%s", sub, opts.TLD)
	}
	if opts.Port > 0 {
		sub = fmt.Sprintf("%s:%d", sub, opts.Port)
	}

	return sub
}

func DecodeHost(host string) (HostInfo, error) {
	info := HostInfo{}

	matches := hostRegex.FindStringSubmatch(host)
	if len(matches) != 6 {
		return HostInfo{}, fmt.Errorf("Couldn't find host in string")
	}

	info.InstanceIP = strings.Replace(matches[1], "-", ".", -1)
	info.SessionId = matches[2]
	info.TLD = matches[4]

	if matches[3] != "" {
		i, _ := strconv.Atoi(matches[3])
		info.EncodedPort = i
	}
	if matches[5] != "" {
		i, _ := strconv.Atoi(matches[5])
		info.Port = i
	}

	return info, nil
}
