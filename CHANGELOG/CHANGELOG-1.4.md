- [v1.4.0](#v140)
  - [Changes since v1.3.1](#changes-since-v131)
    - [Major Themes](#major-themes)
    - [Other notable changes](#other-notable-changes)
    - [Bug Fixes](#bug-fixes)

# v1.4.0

## Changes since v1.3.1

### Major Themes

* Supported control device from north end
* **Replaced Nginx with Envoy**

### Other notable changes

* There was no default TCP Port for each Harns service. It would be assigned automatically from `32100`
* Supported `time series` deletion
* Supported `agent`/`agentType` deletion
* Supported `iotadm add` command that adds API/UI service to Harns

### Bug Fixes

* Fixed `sort` query parameter didn't take effect when querying time series
* Fixed installation name may be duplicated