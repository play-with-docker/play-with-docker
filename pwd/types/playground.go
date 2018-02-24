package types

import (
	"strconv"
	"time"
)

type PlaygroundExtras map[string]interface{}

func (e PlaygroundExtras) Get(name string) (interface{}, bool) {
	v, f := e[name]
	return v, f
}
func (e PlaygroundExtras) GetInt(name string) (int, bool) {
	v, f := e[name]
	if f {
		if r, ok := v.(int); ok {
			return r, ok
		} else if r, ok := v.(float64); ok {
			return int(r), ok
		} else if r, ok := v.(string); ok {
			if v, err := strconv.Atoi(r); err != nil {
				return 0, false
			} else {
				return v, true
			}
		}
		return v.(int), f
	} else {
		return 0, f
	}
}

func (e PlaygroundExtras) GetString(name string) (string, bool) {
	v, f := e[name]
	if f {
		if r, ok := v.(int); ok {
			return strconv.Itoa(r), ok
		} else if r, ok := v.(float64); ok {
			return strconv.FormatFloat(r, 'g', -1, 64), ok
		} else if r, ok := v.(bool); ok {
			return strconv.FormatBool(r), ok
		} else if r, ok := v.(string); ok {
			return r, ok
		} else {
			return "", false
		}
	} else {
		return "", f
	}
}

func (e PlaygroundExtras) GetDuration(name string) (time.Duration, bool) {
	v, f := e[name]
	if f {
		if r, ok := v.(int); ok {
			return time.Duration(r), ok
		} else if r, ok := v.(float64); ok {
			return time.Duration(r), ok
		} else if r, ok := v.(string); ok {
			if d, err := time.ParseDuration(r); err != nil {
				return time.Duration(0), false
			} else {
				return d, true
			}
		} else {
			return time.Duration(0), false
		}
	} else {
		return time.Duration(0), f
	}
}

type Playground struct {
	Id                          string           `json:"id" bson:"id"`
	Domain                      string           `json:"domain" bson:"domain"`
	DefaultDinDInstanceImage    string           `json:"default_dind_instance_image" bson:"default_dind_instance_image"`
	AvailableDinDInstanceImages []string         `json:"available_dind_instance_images" bson:"available_dind_instance_images"`
	AllowWindowsInstances       bool             `json:"allow_windows_instances" bson:"allow_windows_instances"`
	DefaultSessionDuration      time.Duration    `json:"default_session_duration" bson:"default_session_duration"`
	Extras                      PlaygroundExtras `json:"extras" bson:"extras"`
	AssetsDir                   string           `json:"assets_dir" bson:"assets_dir"`
	Tasks                       []string         `json:"tasks" bson:"tasks"`
	GithubClientID              string           `json:"github_client_id" bson:"github_client_id"`
	GithubClientSecret          string           `json:"github_client_secret" bson:"github_client_secret"`
	FacebookClientID            string           `json:"facebook_client_id" bson:"facebook_client_id"`
	FacebookClientSecret        string           `json:"facebook_client_secret" bson:"facebook_client_secret"`
	DockerClientID              string           `json:"docker_client_id" bson:"docker_client_id"`
	DockerClientSecret          string           `json:"docker_client_secret" bson:"docker_client_secret"`
	MaxInstances                int              `json:"max_instances" bson:"max_instances"`
}
