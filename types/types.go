package types

const (
	Ok        = "Ok"
	Fail      = "Fail"
	Deployed  = "Deployed"
	Running   = "Running"
	Stopped   = "Stopped"
	InvalidID = "Application ID not found"
)

type Metadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Applications struct {
	Apps []App `json:"applications"`
}

type App struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type AppDetails struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Containers []Container
}

type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	State   string `json:"state"`
	Ports   string `json:"ports"`
}
