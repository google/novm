Name: novm
Summary: %{summary}
Version: %{version}
Release: %{release}
Group: System
License: ASL 2.0
URL: %{url}
Packager: %{maintainer}
BuildArch: %{architecture}
BuildRoot: %{_tmppath}/%{name}.%{version}-buildroot
Requires: dnsmasq, bridge-utils, fakeroot

# To prevent ypm/rpm/zypper/etc from complaining about FileDigests when
# installing we set the algorithm explicitly to MD5SUM. This should be
# compatible across systems (e.g. RedHat or openSUSE) and is backwards
# compatible.
%global _binary_filedigest_algorithm 1

%description
%{summary}

%install
rm -rf $RPM_BUILD_ROOT
install -d $RPM_BUILD_ROOT
rsync -rav --delete ../../dist/* $RPM_BUILD_ROOT

%files
/usr/bin/novm
/usr/bin/novm-import-kernel
/usr/bin/novm-clear-kernels
/usr/lib/novm

%changelog
* Sat Dec 07 2013 Adin Scannell <adin@scannell.ca>
- Initial package creation.
