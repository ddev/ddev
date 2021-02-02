## User-defined Hosting Provider Integration

Users can define their own provider integration using a simple yaml file in the project .ddev/providers file.

The .ddev/providers directory has a number of examples, including flatfile, platform, pantheon, rsync, and ddev-live examples.

* If general authentication is required, it can be 
    * Added via an global environment variable (as is done with the token in the platform.sh example)
    * Added via a project-level environment variable (in web_environment in the project .ddev/config.yaml or in a .ddev/config.*.yaml)
    * Done via an authentication activity
