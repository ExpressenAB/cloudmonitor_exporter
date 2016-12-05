#!/bin/bash
os=$(uname)
os=${os,,}
arch=$(uname -m)
[[ $arch == "x86_64" ]] && arch="amd64"

package="cloudmonitor_exporter"
echo "Provisioning started, installing packages..."
yum -y install rpmdevtools mock

echo "Setting up rpm dev tree..."
rpmdev-setuptree

echo "Copying files for build..."
cp /docker/package/rpm/$package.spec $HOME/rpmbuild/SPECS/
find /docker/package/sources -type f -exec cp -f {} $HOME/rpmbuild/SOURCES/ \;
cp /docker/build/${package}_${1}_${os}_${arch}/${package} $HOME/rpmbuild/SOURCES/${package}
cd ${HOME}
chown -R root:root rpmbuild
echo "Downloading dependencies..."
spectool -g -R rpmbuild/SPECS/$package.spec

echo "Building rpm..."
rpmbuild -ba --define "_version ${1}" rpmbuild/SPECS/$package.spec

echo "Copying rpms back to build folder...."
cp -f ${HOME}/rpmbuild/RPMS/x86_64/*.rpm /docker/build/rpm/
chmod 777 /docker/build/rpm/*

echo "Successfully built RPM for version ${1}:"
ls -altrh /docker/build/rpm/
exit 0