# AuthProxyWorkload Reference Documentation

Containing important details about how the AuthProxyWorkload resource can
be configured.

## Port and PortEnvName

If HostEnvName is set, the operator will set an EnvVar with the value 127.0.0.1.
If HostEnvName is empty, then no EnvVar is set.

If PortEnvName is set, the operator will set an EnvVar with the port number used
for that
instance. If PortEnvName is empty, then no EnvVar is set.

The port used for an instance is computed by `updateState.useInstancePort()`
which ensures that either Port is used if set, or else a non-conflicting port
number is chosen by the operator.

At least one of Port and PortEnvName must be set for the configuration to be
valid. (We need to add this validation to the operator. It will be handled in
authproxyworkload_webhook.go)

This is how Port and PortEnvName should interact:

| PortEnvName      | Port        | proxy port args | container env  |    
|------------------|-------------|-----------------|----------------|
| set to "DB_PORT" | not set     | ?port={next}    | DB_PORT={next} |
| set to "DB_PORT" | set to 5401 | ?port=5401      | DB_PORT=5401.  |
| not set          | set to 5401 | ?port=5401      | not set        |
| not set          | not set     | invalid.        | invalid        |


 