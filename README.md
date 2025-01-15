# TaskBalancerX


# Distributed Task Processing System

## Overview
This project is a distributed task processing system consisting of:
1. **Go Server**: Accepts tasks (files) and queues them for processing.
2. **Rust Worker**: Processes tasks asynchronously.

## Features
- Graceful shutdown.
- Task status tracking.
- Health checks for monitoring.
- Retry logic for task processing.

## Requirements
- Go 1.20+
- Rust 1.72+

## Setup
1. Clone the repository.
2. Navigate to `go-task-queue` and `rust-worker` directories to build and run each service.

### Go Server
```bash
cd go-task-queue
go mod tidy
go run main.go
