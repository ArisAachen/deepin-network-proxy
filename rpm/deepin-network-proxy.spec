%global debug_package   %{nil}

Name:           deepin-network-proxy
Version:        1.0.0.7
Release:        1
Summary:        Most simple RPM package
License:        GPLv3

Source0:        %{name}-%{version}.orig.tar.xz

BuildRequires:  compiler(go-compiler)
BuildRequires:  pkgconfig(gdk-3.0)
BuildRequires:  pkgconfig(glib-2.0)
BuildRequires:  pkgconfig(gobject-2.0)
BuildRequires:  gocode
BuildRequires:  golang-github-linuxdeepin-go-x11-client-devel
BuildRequires:  golang-github-linuxdeepin-go-dbus-factory-devel
BuildRequires:  go-lib-devel
BuildRequires:  go-gir-generator

%description
This is my first RPM package, which does nothing.

%prep
%autosetup
patch -p1 < rpm/lib-to-libexec.patch

%build
export GOPATH=/usr/share/gocode
%make_build GO_BUILD_FLAGS=-trimpath

%install
%make_install

%files
%config %{_sysconfdir}/deepin/deepin-proxy/*
%{_datadir}/dbus-1/system.d/*
%{_datadir}/dbus-1/system-services/*
%{_libexecdir}/deepin-daemon/*

%changelog
# let's skip this for now
