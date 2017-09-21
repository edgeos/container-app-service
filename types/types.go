package types

//Constants ...
const (
	Ok        = "Ok"
	Fail      = "Fail"
	Deployed  = "Deployed"
	Running   = "Running"
	Stopped   = "Stopped"
	InvalidID = "Application ID not found"
)

//Metadata ...
type Metadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

//Applications ...
type Applications struct {
	Apps []App `json:"applications"`
}

//App ...
type App struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

//AppDetails ...
type AppDetails struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Containers []Container
}

//Container ...
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	State   string `json:"state"`
	Ports   string `json:"ports"`
}
