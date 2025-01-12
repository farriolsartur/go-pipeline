# go-pipeline

A flexible Go library that allows you to build and run a series of **steps** (functions) in a configurable pipeline. It supports:
* Dynamically reordering step execution.
* Multiple argument resolution policies (by type-based rolling index or fail if missing).
* Custom argument bindings (from initial inputs or previous step outputs). 
* A configurable logger (using **Logrus**) at both the global and pipeline levels.

This library is specifically tailored for applications that reuse the same functions across different processes or algorithms.

## Installation

Coming soon...

## Quick start

Coming soon

## Future additions

1. Enable passing structs to execute interface functions.
2. Improved configuration options and simplified syntax.
3. Implement parallel processing capabilities.