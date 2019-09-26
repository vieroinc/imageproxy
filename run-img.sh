# local img > api comm is http, not https!
ADDR=${1:-localhost:8083}
STORAGE=${2:-/mnt/vdos/imageproxy}
BASEURL=${3:-http://api.vierodev.tv:8082/v2/node/}
CGO_CFLAGS_ALLOW="-Xpreprocessor" go build -o main ./cmd/imageproxy
./main -addr ${ADDR} -cache memory -baseURL ${BASEURL}

