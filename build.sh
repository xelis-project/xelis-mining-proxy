# make sure you install xgo using:
# go get src.techknowlogick.com/xgo

rm build -r

mkdir build
cp LICENSE.txt ./build/
cp README.md ./build/
cd build/

xgo -trimpath -ldflags="-s -w" --targets linux/amd64,windows-6.1/amd64,darwin-10.12/amd64 ..

# rename windows file
mv xelis-mining-proxy-windows-6.1-amd64.exe xelis-mining-proxy-windows-amd64.exe
# rename darwin file
mv xelis-mining-proxy-darwin-10.12-amd64 xelis-mining-proxy-darwin-amd64

GZIP=-9 tar --xz -cf xelis-mining-proxy-linux-amd64.tar.xz xelis-mining-proxy-linux-amd64 LICENSE.txt README.md
GZIP=-9 tar --xz -cf xelis-mining-proxy-darwin-amd64.tar.xz xelis-mining-proxy-darwin-amd64 LICENSE.txt README.md
zip -9 xelis-mining-proxy-windows-amd64.zip xelis-mining-proxy-windows-amd64.exe LICENSE.txt README.md
