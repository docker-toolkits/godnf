cd ../
sh -x build.sh
cd -
cp ../godnf .
docker build -t opencloudos-godnf -f dockerfile.ocs .
docker build -t ubi9-godnf -f dockerfile.rh .