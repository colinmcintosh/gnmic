### Description

The `[subscribe | sub]` command represents the [gNMI Subscribe RPC](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto#L68).

It is used to send a [Subscribe Request](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto#L208) to the specified target(s) and expects one or multiple [Subscribe Response](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto#L232)

### Usage
`gnmic [global-flags] subscribe [local-flags]`

### Local Flags
The subscribe command supports the following local flags:

#### prefix
The `[--prefix]` flag sets a common [prefix](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#241-path-prefixes) to all paths specified using the local `--path` flag. Defaults to `""`.

#### path
The path flag `[--path]` is used to specify the [path(s)](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#222-paths) to which the client wants to subscribe.

Multiple paths can be specified by using repeated `--path` flags:

```bash
gnmic sub --path "/state/ports[port-id=*]" \
          --path "/state/router[router-name=*]/interface[interface-name=*]"
```

If a user needs to provide [origin](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#222-paths) information to the Path message, the following pattern should be used for the path string: `"origin:path"`:

```
gnmic sub --path "openconfig-interfaces:interfaces/interface"
```

#### target
With the optional `[--target]` flag it is possible to supply the [path target](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#2221-path-target) information in the prefix field of the SubscriptionList message.

#### model
The `[--model]` flag is used to specify the schema definition modules that the target should use when extracting the data to stream back.

#### qos
The `[--qos]` flag specifies the packet marking that is to be used for the responses to the subscription request. Default marking is set to `20`. If qos marking is not supported by a target the marking can be disabled by setting the value to `0`.

#### mode
The `[--mode]` mode flag specifies the mode of subscription to be created.

This may be one of:
[ONCE](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35151-once-subscriptions), [STREAM](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35152-stream-subscriptions) or [POLL](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35153-poll-subscriptions).

It is case insensitive and defaults to `STREAM`.

#### stream subscription mode
The `[--stream-mode]` flag is used to specify the stream subscription mode.

This may be one of: [ON_CHANGE, SAMPLE or TARGET_DEFINED](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35152-stream-subscriptions)

This flag applies only if `--mode` is set to `STREAM`. It is case insensitive and defaults to `SAMPLE`.

#### sample interval
The `[--sample-interval]` flag is used to specify the sample interval to be used by the target to send samples to the client.

This flag applies only in case `--mode` is set to `STREAM` and `--stream-mode` is set to `SAMPLE`.

Valid formats: `1s, 1m30s, 1h`. Defaults to `0s` which is the lowest interval supported by a target.

#### heartbeat interval
The `[--heartbeat-interval]` flag is used to specify the server heartbeat interval.

The heartbeat interval value can be specified along with `ON_CHANGE` or `SAMPLE` stream subscriptions modes.

* `ON_CHANGE`: The value of the data item(s) MUST be re-sent once per heartbeat interval regardless of whether the value has changed or not.
* `SAMPLE`: The target MUST generate one telemetry update per heartbeat interval, regardless of whether the `--suppress-redundant` flag is set to true.

#### quiet
With `[--quiet]` flag set `gnmic` will not output subscription responses to `stdout`. The `--quiet` flag is useful when `gnmic` exports the received data to one of the export providers.

#### suppress redundant
When the `[--suppress-redundant]` flag is set to true, the target SHOULD NOT generate a telemetry update message unless the value of the path being reported on has changed since the last update was generated.

This flag applies only in case `--mode` is set to `STREAM` and `--stream-mode` is set to `SAMPLE`.

#### updates only
When the `[--updates-only]` flag is set to true, the target MUST not transmit the current state of the paths that the client has subscribed to, but rather should send only updates to them.

#### name
The `[--name]` flag is used to trigger one or multiple subscriptions already defined in the configuration file see [defining subscriptions](../advanced/subscriptions.md)

#### output
The `[--output]` flag is used to select one or multiple output already defined in the configuration file. 

Outputs defined under target take precedence over this flag, see [defining outputs](../advanced/multi_outputs/output_intro.md) and [defining targets](../advanced/multi_targets)

### Examples
#### 1. streaming, target-defined, 10s interval
```bash
gnmic -a <ip:port> sub --path /state/port[port-id=*]/statistics
```

#### 2. streaming, sample, 30s interval
```bash
gnmic -a <ip:port> sub --path "/state/port[port-id=*]/statistics" \
                       --sample-interval 30s
```

#### 3. streaming, on-change, heartbeat interval 1min
```bash
gnmic -a <ip:port> sub --path "/state/port[port-id=*]/statistics" \
                       --stream-mode on-change \
                       --heartbeat-interval 1m
```

#### 4. once subscription
```bash
gnmic -a <ip:port> sub --path "/state/port[port-id=*]/statistics" \
                       --mode once
```

<script
id="asciicast-319608" src="https://asciinema.org/a/319608.js" async>
</script>
