Name:           cloudmonitor_exporter
Version:        %{_version}
Release:        1
Summary:        Prometheus exporter for Akamai Cloudmonitor.
Group:          System Environment/Daemons
License:        Apache Software License
URL:            https://github.com/ExpressenAB/cloudmonitor_exporter
Source0:        %{name}
Source1:        %{name}.service
Source2:        %{name}.sysconfig
BuildRoot:      %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)

%description
Prometheus exporter for Akamai Cloudmonitor.

%install
mkdir -p %{buildroot}/%{_sbindir}
cp %{SOURCE0} %{buildroot}/%{_sbindir}/%{name}

mkdir -p %{buildroot}/%{_sysconfdir}/sysconfig
cp %{SOURCE2} %{buildroot}/%{_sysconfdir}/sysconfig/%{name}

mkdir -p %{buildroot}/%{_unitdir}
cp %{SOURCE1} %{buildroot}/%{_unitdir}/

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}
%{_unitdir}/%{name}.service
%attr(755, root, root) %{_sbindir}/*

%doc


%changelog
* Mon Dec 4 2016 Rickard Karlsson <rickard.karlsson@bonniernews.se>
- Release 0.1.3
