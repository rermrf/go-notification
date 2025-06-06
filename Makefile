
# buf 生成
.PHONY: grpc
buf:
	@buf format -w api/proto
	@buf lint api/proto
	@buf generate api/proto

