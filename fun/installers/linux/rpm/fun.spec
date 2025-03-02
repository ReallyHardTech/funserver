Name:           fun
Version:        VERSION
Release:        1%{?dist}
Summary:        Fun Server - Docker Container Manager

License:        Apache-2.0
URL:            https://thefunserver.com
Source0:        %{name}-%{version}.tar.gz

Requires:       docker-ce >= 19.0.0, systemd
BuildRequires:  systemd

%description
Fun Server manages your local funserver installation, allowing you to install
and manage compatible applications seamlessly. It supports installation on
macOS, Windows, and Linux.

Features:
* Manage your local funserver installation
* Install and manage compatible applications
* Cross-platform support (macOS, Windows, Linux)

%prep
%setup -q

%install
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/etc/systemd/system
mkdir -p %{buildroot}/%{_mandir}/man1
mkdir -p %{buildroot}/%{_datadir}/applications
install -m 755 %{name} %{buildroot}/usr/local/bin/%{name}
install -m 644 fun.service %{buildroot}/etc/systemd/system/fun.service
install -m 644 man/fun.1 %{buildroot}/%{_mandir}/man1/fun.1
install -m 644 fun.desktop %{buildroot}/%{_datadir}/applications/fun.desktop
gzip -9 %{buildroot}/%{_mandir}/man1/fun.1

%post
# Create configuration directory if it doesn't exist
mkdir -p /etc/fun
mkdir -p /var/log/fun

# Create default configuration if it doesn't exist
if [ ! -f /etc/fun/config.json ]; then
    cat > /etc/fun/config.json << EOF
{
  "cloud_url": "https://api.thefunserver.com",
  "poll_interval": 60,
  "docker_host": "unix:///var/run/docker.sock",
  "docker_network": "fun_network",
  "log_level": "info",
  "log_file": "/var/log/fun/fun.log"
}
EOF
    chmod 600 /etc/fun/config.json
fi

# Reload systemd configuration
systemctl daemon-reload

# Enable and start the service
systemctl enable fun
systemctl start fun || true

# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q
fi

%preun
# Stop and disable the service
systemctl stop fun || true
systemctl disable fun || true

# Reload systemd configuration
systemctl daemon-reload

%postun
# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q
fi

%files
%defattr(-,root,root,-)
/usr/local/bin/%{name}
/etc/systemd/system/fun.service
%{_mandir}/man1/fun.1.gz
%{_datadir}/applications/fun.desktop
%license LICENSE
%doc README.md

%changelog
* Tue May 21 2024 Fun Server Team <support@thefunserver.com> - VERSION-1
- Initial package 