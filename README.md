# Protohackers in Go

## Build docker images
1. Navigate to repo root
2. Execute command ```docker build -f Dockerfile.{app_folder} --platform linux/arm64 -t {container_registry}/{app_folder}:latest .```

## Push docker images
1. Navigate to repo root
2. Execute command ```docker push {container_registry}/{app_folder}:latest```