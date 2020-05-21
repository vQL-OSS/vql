%define vqld_user_uid     811
%define vqld_user_gid     %{vqld_user_uid}

Name:		vqld
Version:	1.0
Release:	0
Summary:	vqld - Virtual Queue Line Service.
Group:		Applications/Internet
License:	MIT
Vendor:		FurtherSystem Co.,Ltd.
URL:		http://vqloss.info/
Source0:	%{name}-%{version}-%{release}.x86_64.tar.gz
Requires(pre,postun):  %{_sbindir}/groupadd
Requires(pre,postun):  %{_sbindir}/useradd
Requires(pre,post,preun,postun):  %{_bindir}/systemctl

%description
vQLd ... virtual Queue Line Service.

%global debug_package %{nil}

%prep
rm -rf %{buildroot}

%setup -n %{name}-%{version}-%{release}.x86_64

%build

%install
install -d %{buildroot}/var/log/%{name} %{buildroot}/usr/local/%{name}/bin/ %{buildroot}%{_sysconfdir}/sysconfig/ %{buildroot}%{_sysconfdir}/systemd/system/
install %{name} %{buildroot}/usr/local/%{name}/bin/
install %{name}-boot.sh %{buildroot}/usr/local/%{name}/bin/
install -S .rpmsave -b %{name}.env %{buildroot}%{_sysconfdir}/sysconfig/
install %{name}.service %{buildroot}%{_sysconfdir}/systemd/system/

%clean
rm -rf %{buildroot}

%pre
if [ $1 -eq 1 ] ; then
    %{_sbindir}/groupadd -g %{vqld_user_gid} vqld_user >/dev/null 2>&1 || :
    %{_sbindir}/useradd -u %{vqld_user_uid} -s /sbin/nologin -g vqld_user vqld_user >/dev/null 2>&1 || :
fi

%post
if [ $1 -eq 1 ] ; then
    %{_bindir}/systemctl daemon-reload >/dev/null 2>&1 || :
fi

%preun
if [ $1 -eq 0 ] ; then
    %{_bindir}/systemctl --no-reload disable %{name}.service > /dev/null 2>&1 || :
    %{_bindir}/systemctl stop %{name}.service > /dev/null 2>&1 || :
fi

%postun
%{_bindir}/systemctl daemon-reload >/dev/null 2>&1 || :
if [ $1 -ge 1 ] ; then
    %{_bindir}/systemctl try-restart %{name}.service >/dev/null 2>&1 || :
elif [ $1 -eq 0 ] ; then
    %{_sbindir}/userdel vqld_user >/dev/null 2>&1 || :
    %{_sbindir}/groupdel vqld_user >/dev/null 2>&1 || :
fi

%files
%defattr(0644, root, root, -)
%{_sysconfdir}/systemd/system/%{name}.service
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}.env
%defattr(0755, vqld_user, vqld_user, 0755)
/usr/local/%{name}/bin/%{name}
/usr/local/%{name}/bin/%{name}-boot.sh
/var/log/%{name}
%license LICENSE

%changelog

