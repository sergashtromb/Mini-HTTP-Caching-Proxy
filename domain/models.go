// this file needed to define a structs

package domain

type StartSettings struct {
	ConfigName string
	GenConfigFile bool
}

type InboxRequest struct {
	Method 	string
	IP 		string
	Port 	string
	Host 	string
	Path 	string
}