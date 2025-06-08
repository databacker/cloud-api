openapi: generate client and server code from api.yaml

What does the API need to include? Condensed list here, to be translated to api.yaml after.

resources: instance, config, log

- create a new instance (admin)
- create a device config (admin)
- get a list of instances (admin)
- get info about a specific device, other than config (admin) 
- get information about a backup attempt, e.g. date, device, start, finish, etc., except for actual logs (admin)
- get logs for a backup attempt (admin)
