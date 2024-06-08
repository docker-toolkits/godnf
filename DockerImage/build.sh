cd ../
sh -x build.sh
cd -
cp ../godnf .
docker build -t ocs23 -f dockerfile.ocs23 .
docker build -t ubi9 -f dockerfile.ubi9 .