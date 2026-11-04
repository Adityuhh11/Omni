package palette

// typedForSystemRows is what a person types to reach each system row on Windows. Sleep and Hibernate are left out: a computer
// without them (a virtual machine in CI) does not offer them.
var typedForSystemRows = map[string]string{
	"lock": "sys:lock", "task man": "sys:taskmgr", "recycle": "sys:recycle", "windows set": "sys:winsettings",
	"shut": "sys:shutdown", "restart": "sys:restart", "sign out": "sys:signout", "display": "sys:displayoff",
}
