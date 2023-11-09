# Go Database Reconciler

A library for translating between YAML/JSON-serializable Golang structs and
Kong databases. It can read the Kong HTTP admin API and create a struct
representation of the current configuration set and update a Kong instance's
configuration to match a configuration struct.
