- [v1.5.2](#v152)
  - [Changes since v1.5.1](#changes-since-v151)
    - [Major Themes](#major-themes)
    - [Bug Fixes](#bug-fixes)
- [v1.5.1](#v151)
  - [Changes since v1.5.0](#changes-since-v150)
    - [Major Themes](#major-themes-1)
    - [Other notable changes](#other-notable-changes)
    - [Bug Fixes](#bug-fixes-1)
- [v1.5.0](#v150)
  - [Changes since v1.4.0](#changes-since-v140)
    - [Major Themes](#major-themes-2)
    - [Other notable changes](#other-notable-changes-1)
    - [Bug Fixes](#bug-fixes-2)

# v1.5.2

## Changes since v1.5.1

### Major Themes

* Supported `thing`/`thingType`/`propertySetType` deletion
* Supported rollup on the fly
* Supported `command history` deletion after `commandType` deletion

### Other notable changes

* Supported gateway rewrite host when the cluster is not local cluster

### Bug Fixes

* Fixed a bug where notification manager deleted template if it contained other message template after deletion a message template
* Fixed a bug where rollup persisted pre-aggregated boolean property even if its value was null
* Fixed a bug where failed updating agent if it generated from an agent type
* Fixed a bug where couldn't find mapping if restart iot-data-broker without breaking upload data

# v1.5.1

## Changes since v1.5.0

### Major Themes

* Supported `event` deletion
* Supported `eventType` deletion
* Supported `event` storage on cloud
  
### Other notable changes

* Supported query the hierarchical events

### Bug Fixes

* Fixed a bug where event manager failed querying events crossed TTL
* Fixed a bug where the result of query rollup did NOT sort by time
* Fixed a bug where the rollup of bool property was not correct
* Fixed a bug where rollup return 25 records if query range is 1 day and interval is 1 hour


# v1.5.0

## Changes since v1.4.0

### Major Themes

* Supported Rollup and Pre-aggregate time series data
* Upgraded influxdb from 1.8.5 to 2.1.1

### Other notable changes

* Supported `iotadm delete` command that deletes API/UI service from Harns
* Supported `rule` deletion
* Supported `commandType` deletion
* Supported `template` deletion
* Supported `messageTemplate` deletion
* Supported `recipient` deletion
* The `time series`/`events`/`command history`/`action history` were **dropped** when upgrading to 1.5.0. Because the data models were adjusted to match influxdb2
* The name of property and characteristic should **NOT** includes `.`

### Bug Fixes

* Fixed a bug where failed saving event when two events have the same TTL, and they have the same field with the different datatype
* Fixed a bug where rule missed execution when an expression has more than one operand
* Fixed a bug on poorly configured machine where mqtt topic triggered racing condition
