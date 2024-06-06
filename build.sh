#yum install upx golang

## -ldflags="-s -w"  Delete the symbol to reduce the size
CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" -o godnf ./cmd/godnf

##Compression to reduce volume
upx -9 godnf
#go build -mod=vendor -ldflags="-s -w -extldflags '-static -lc -ldl'"  -o godnf ./cmd/godnf