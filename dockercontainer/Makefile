gen:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto


build-vendor:
	go mod vendor
