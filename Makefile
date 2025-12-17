generate:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    internal/pb/manager.proto

generate_python:
	python -m grpc_tools.protoc \
    -I internal/pb \
    --python_out=./python/src/acsm_spark \
    --pyi_out=./python/src/acsm_spark \
    --grpc_python_out=./python/src/acsm_spark \
    internal/pb/manager.proto
