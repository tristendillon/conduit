module my-app

go 1.25.0

require github.com/tristendillon/conduit v0.0.0

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/tristendillon/conduit => ../ // this is a placeholder for the actual version of the conduit package
