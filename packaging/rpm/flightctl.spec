# Disable debug information package creation
%define debug_package %{nil}

# Define the Go Import Path
%global goipath github.com/flightctl/flightctl

# SELinux specifics
%global selinuxtype targeted
%define selinux_policyver 3.14.3-67
%define agent_relabel_files() \
    semanage fcontext -a -t flightctl_agent_exec_t "/usr/bin/flightctl-agent" ; \
    restorecon -v /usr/bin/flightctl-agent

Name:           flightctl
Version:        0.6.0
Release:        1%{?dist}
Summary:        Flight Control service

%gometa

License:        Apache-2.0 AND BSD-2-Clause AND BSD-3-Clause AND ISC AND MIT
URL:            %{gourl}

Source0:        1%{?dist}

BuildRequires:  golang
BuildRequires:  make
BuildRequires:  git
BuildRequires:  openssl-devel

Requires: openssl

# Skip description for the main package since it won't be created
%description
# Main package is empty and not created.

# cli sub-package
%package cli
Summary: Flight Control CLI
%description cli
flightctl is the CLI for controlling the Flight Control service.

# agent sub-package
%package agent
Summary: Flight Control management agent

Requires: flightctl-selinux = %{version}
Requires: bootc

%description agent
The flightctl-agent package provides the management agent for the Flight Control fleet management service.

# selinux sub-package
%package selinux
Summary: SELinux policies for the Flight Control management agent
BuildRequires: selinux-policy >= %{selinux_policyver}
BuildRequires: selinux-policy-devel >= %{selinux_policyver}
BuildArch: noarch
Requires: selinux-policy >= %{selinux_policyver}

%description selinux
The flightctl-selinux package provides the SELinux policy modules required by the Flight Control management agent.

# services sub-package
%package services
Summary: Flight Control services
Requires: bash
Requires: podman

%description services
The flightctl-services package provides installation and setup of files for running containerized Flight Control services

%package observability
Summary: Optional observability RPM
Requires:       podman
Requires:       systemd
Requires(post): systemd, yq, gettext
Requires(preun):systemd
Requires(postun):systemd
Requires:       selinux-policy-targeted

%description observability
This package provides the Flightctl Observability Stack, including
Prometheus for metric storage, Grafana for visualization, and
OpenTelemetry Collector for metric collection. All components run in Podman containers
managed by systemd.


%files observability
/etc/containers/systemd/flightctl-prometheus.container
/etc/containers/systemd/flightctl-otel-collector.container

# Static configuration files
/etc/prometheus/prometheus.yml
/etc/otelcol/otelcol-config.yaml

# Central customizable configuration file
/etc/flightctl/observability_config.yaml

# Template source files (installed to a temporary location)
/opt/flightctl-observability/templates/grafana.ini.template
/opt/flightctl-observability/templates/flightctl-grafana.container.template

# Non-templated Grafana datasource provisioning file
/etc/grafana/provisioning/datasources/prometheus.yaml

# The files that will be generated in %post must be listed as %ghost files.
%ghost /etc/grafana/grafana.ini
%ghost /etc/containers/systemd/flightctl-grafana.container

# New reloader script and systemd units
/usr/local/bin/flightctl-observability-config-reloader.sh
/usr/lib/systemd/system/flightctl-observability-config-reloader.service

# Directories owned by the RPM (will be created automatically if they don't exist)
%dir /etc/prometheus
%dir /etc/otelcol
%dir /etc/grafana
%dir /etc/grafana/provisioning
%dir /etc/grafana/provisioning/datasources
%dir /var/lib/prometheus
%dir /var/lib/grafana
%dir /var/lib/otelcol
%dir /etc/flightctl
%dir /opt/flightctl-observability/templates
%dir /usr/local/bin


%post observability
# This script runs AFTER the files have been installed onto the system.
echo "Running post-install actions for Flightctl Observability Stack..."

# Create necessary directories on the host if they don't already exist.
/usr/bin/mkdir -p /etc/prometheus /var/lib/prometheus
/usr/bin/mkdir -p /etc/otelcol /var/lib/otelcol
/usr/bin/mkdir -p /etc/grafana /etc/grafana/provisioning /etc/grafana/provisioning/datasources /var/lib/grafana
/usr/bin/mkdir -p /etc/flightctl /opt/flightctl-observability/templates
/usr/bin/mkdir -p /usr/local/bin /usr/lib/systemd/system

chown 65534:65534 /var/lib/prometheus
chown 472:472 /var/lib/grafana

# Apply persistent SELinux contexts for volumes and configuration files.
/usr/sbin/semanage fcontext -a -t container_file_t "/etc/prometheus/prometheus.yml" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -a -t container_file_t "/var/lib/prometheus(/.*)?" >/dev/null 2>&1 || :

/usr/sbin/semanage fcontext -a -t container_file_t "/etc/grafana(/.*)?" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -a -t container_file_t "/var/lib/grafana(/.*)?" >/dev/null 2>&1 || :

/usr/sbin/semanage fcontext -a -t container_file_t "/etc/otelcol/otelcol-config.yaml" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -a -t container_file_t "/var/lib/otelcol(/.*)?" >/dev/null 2>&1 || :

/usr/sbin/semanage fcontext -a -t container_file_t "/opt/flightctl-observability/templates(/.*)?" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -a -t container_file_t "/usr/local/bin/flightctl-observability-config-reloader.sh" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -a -t container_file_t "/usr/lib/systemd/system/flightctl-observability-config-reloader.service" >/dev/null 2>&1 || :

# Restore file contexts based on the new rules (and default rules)
/usr/sbin/restorecon -RvF /etc/prometheus >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/prometheus >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /etc/grafana >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/grafana >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /etc/otelcol >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/otelcol >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /opt/flightctl-observability/templates >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /usr/local/bin/flightctl-observability-config-reloader.sh >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /usr/lib/systemd/system/flightctl-observability-config-reloader.service >/dev/null 2>&1 || :

# Enable specific SELinux boolean if needed
/usr/sbin/setsebool -P container_manage_cgroup on >/dev/null 2>&1 || :


# --- Process Configuration Templates (Initial Generation) ---
# Call the reloader script once during installation to generate initial config files.
echo "Generating initial configuration files using the reloader script..."
/usr/local/bin/flightctl-observability-config-reloader.sh || { echo "ERROR: Initial config generation failed!"; exit 1; }


# --- Final service management ---
echo "Reloading systemd daemon..."
/usr/bin/systemctl daemon-reload

echo "Enabling and starting core services..."
/usr/bin/systemctl enable flightctl-prometheus.service
/usr/bin/systemctl start flightctl-prometheus.service

/usr/bin/systemctl start flightctl-otel-collector.service

# Start Grafana after initial templating (it will be restarted by reloader if config changes)
/usr/bin/systemctl start flightctl-grafana.service

# Enable and start the config reloader path unit.
# This makes Grafana's config dynamically update on config file changes.

echo "Flightctl Observability Stack services installed and started. Grafana config is now dynamic."




%preun observability
echo "Running pre-uninstall actions for Flightctl Observability Stack..."
# Stop and disable all services in reverse order of startup
/usr/bin/systemctl stop flightctl-grafana.service >/dev/null 2>&1 || :
/usr/bin/systemctl disable flightctl-grafana.service >/dev/null 2>&1 || :

/usr/bin/systemctl stop flightctl-observability-config-reloader.service >/dev/null 2>&1 || :
/usr/bin/systemctl disable flightctl-observability-config-reloader.service >/dev/null 2>&1 || :

/usr/bin/systemctl stop flightctl-otel-collector.service >/dev/null 2>&1 || :
/usr/bin/systemctl disable flightctl-otel-collector.service >/dev/null 2>&1 || :

/usr/bin/systemctl stop flightctl-prometheus.service >/dev/null 2>&1 || :
/usr/bin/systemctl disable flightctl-prometheus.service >/dev/null 2>&1 || :


%postun observability
echo "Running post-uninstall actions for Flightctl Observability Stack..."
# Clean up Podman containers associated with the services
/usr/bin/podman rm -f flightctl-grafana >/dev/null 2>&1 || :
/usr/bin/podman rm -f flightctl-otel-collector >/dev/null 2>&1 || :
/usr/bin/podman rm -f flightctl-prometheus >/dev/null 2>&1 || :

# Remove SELinux fcontext rules added by this package
/usr/sbin/semanage fcontext -d -t container_file_t "/etc/grafana(/.*)?" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/var/lib/grafana(/.*)?" >/dev/null 2>&1 || :

/usr/sbin/semanage fcontext -d -t container_file_t "/etc/prometheus/prometheus.yml" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/var/lib/prometheus(/.*)?" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/etc/otelcol/otelcol-config.yaml" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/var/lib/otelcol(/.*)?" >/dev/null 2>&1 || :

/usr/sbin/semanage fcontext -d -t container_file_t "/opt/flightctl-observability/templates(/.*)?" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/usr/local/bin/flightctl-observability-config-reloader.sh" >/dev/null 2>&1 || :
/usr/sbin/semanage fcontext -d -t container_file_t "/usr/lib/systemd/system/flightctl-observability-config-reloader.service" >/dev/null 2>&1 || :


# Restore default SELinux contexts for affected directories
/usr/sbin/restorecon -RvF /etc/grafana >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/grafana >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /etc/prometheus >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/prometheus >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /etc/otelcol >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /var/lib/otelcol >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /opt/flightctl-observability/templates >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /usr/local/bin/flightctl-observability-config-reloader.sh >/dev/null 2>&1 || :
/usr/sbin/restorecon -RvF /usr/lib/systemd/system/flightctl-observability-config-reloader.service >/dev/null 2>&1 || :


/usr/bin/systemctl daemon-reload
echo "Flightctl Observability Stack uninstalled."

%prep
%goprep -A
%setup -q %{forgesetupargs}

%build
    # if this is a buggy version of go we need to set GOPROXY as workaround
    # see https://github.com/golang/go/issues/61928
    GOENVFILE=$(go env GOROOT)/go.env
    if [[ ! -f "${GOENVFILE}" ]]; then
        export GOPROXY='https://proxy.golang.org,direct'
    fi

    SOURCE_GIT_TAG=$(echo %{version} | tr '~' '-') \
    SOURCE_GIT_TREE_STATE=clean \
    SOURCE_GIT_COMMIT=$(echo %{version} | awk -F'[-~]g' '{print $2}') \
    SOURCE_GIT_TAG_NO_V=%{version} \
    make build-cli build-agent

    # SELinux modules build
    make --directory packaging/selinux

%install
    mkdir -p %{buildroot}/usr/bin
    mkdir -p %{buildroot}/etc/flightctl
    cp bin/flightctl %{buildroot}/usr/bin
    mkdir -p %{buildroot}/usr/lib/systemd/system
    mkdir -p %{buildroot}/%{_sharedstatedir}/flightctl
    mkdir -p %{buildroot}/usr/lib/flightctl/custom-info.d
    mkdir -p %{buildroot}/usr/lib/flightctl/hooks.d/{afterupdating,beforeupdating,afterrebooting,beforerebooting}
    mkdir -p %{buildroot}/usr/lib/greenboot/check/required.d
    install -m 0755 packaging/greenboot/flightctl-agent-running-check.sh %{buildroot}/usr/lib/greenboot/check/required.d/20_check_flightctl_agent.sh
    cp bin/flightctl-agent %{buildroot}/usr/bin
    cp packaging/must-gather/flightctl-must-gather %{buildroot}/usr/bin
    cp packaging/hooks.d/afterupdating/00-default.yaml %{buildroot}/usr/lib/flightctl/hooks.d/afterupdating
    cp packaging/systemd/flightctl-agent.service %{buildroot}/usr/lib/systemd/system
    bin/flightctl completion bash > flightctl-completion.bash
    install -Dpm 0644 flightctl-completion.bash -t %{buildroot}/%{_datadir}/bash-completion/completions
    bin/flightctl completion fish > flightctl-completion.fish
    install -Dpm 0644 flightctl-completion.fish -t %{buildroot}/%{_datadir}/fish/vendor_completions.d/
    bin/flightctl completion zsh > _flightctl-completion
    install -Dpm 0644 _flightctl-completion -t %{buildroot}/%{_datadir}/zsh/site-functions/
    install -d %{buildroot}%{_datadir}/selinux/packages/%{selinuxtype}
    install -m644 packaging/selinux/*.bz2 %{buildroot}%{_datadir}/selinux/packages/%{selinuxtype}

    rm -f licenses.list

    find . -type f -name LICENSE -or -name License | while read LICENSE_FILE; do
        echo "%{_datadir}/licenses/%{NAME}/${LICENSE_FILE}" >> licenses.list
    done
    mkdir -vp "%{buildroot}%{_datadir}/licenses/%{NAME}"
    cp LICENSE "%{buildroot}%{_datadir}/licenses/%{NAME}"

    mkdir -vp "%{buildroot}%{_docdir}/%{NAME}"

    for DOC in docs examples .markdownlint-cli2.yaml README.md; do
        cp -vr "${DOC}" "%{buildroot}%{_docdir}/%{NAME}/${DOC}"
    done

    # flightctl-services sub-package steps
    # Run the install script to move the quadlet files.
    #
    # The IMAGE_TAG is derived from the RPM version, which may include tildes (~)
    # for proper version sorting (e.g., 0.5.1~rc1-1). However, the tagged images
    # always use hyphens (-) instead of tildes (~). To ensure valid image tags we need
    # to transform the version string by replacing tildes with hyphens.
    CONFIG_READONLY_DIR="%{buildroot}%{_datadir}/flightctl" \
    CONFIG_WRITEABLE_DIR="%{buildroot}%{_sysconfdir}/flightctl" \
    QUADLET_FILES_OUTPUT_DIR="%{buildroot}%{_datadir}/containers/systemd" \
    SYSTEMD_UNIT_OUTPUT_DIR="%{buildroot}/usr/lib/systemd/system" \
    IMAGE_TAG=$(echo %{version} | tr '~' '-') \
    deploy/scripts/install.sh

    # Copy sos report flightctl plugin
    mkdir -p %{buildroot}/usr/share/sosreport
    cp packaging/sosreport/sos/report/plugins/flightctl.py %{buildroot}/usr/share/sosreport

    # install observability
     # Create target directories within the build root (where files are staged for RPM)
     mkdir -p %{buildroot}/etc/containers/systemd
     mkdir -p %{buildroot}/etc/prometheus
     mkdir -p %{buildroot}/etc/otelcol
     mkdir -p %{buildroot}/etc/grafana/provisioning/datasources # For Grafana's datasource config
     mkdir -p %{buildroot}/var/lib/prometheus
     mkdir -p %{buildroot}/var/lib/grafana # For Grafana's data
     mkdir -p %{buildroot}/var/lib/otelcol
     mkdir -p %{buildroot}/opt/flightctl-observability/templates # Staging for template files processed in %post
     mkdir -p %{buildroot}/usr/local/bin # For the reloader script
     mkdir -p %{buildroot}/usr/lib/systemd/system # For systemd units
     
     # Copy static Quadlet files (those not templated)
     install -m 0644 packaging/observability/flightctl-prometheus.container %{buildroot}/etc/containers/systemd/
     install -m 0644 packaging/observability/flightctl-otel-collector.container %{buildroot}/etc/containers/systemd/
     
     # Copy static configuration files (those not templated)
     install -m 0644 packaging/observability/prometheus.yml %{buildroot}/etc/prometheus/
     install -m 0644 packaging/observability/otelcol-config.yaml %{buildroot}/etc/otelcol/
     
     # Copy the central customizable config YAML
     install -m 0644 packaging/observability/observability_config.yaml %{buildroot}/etc/flightctl/
     
     # Copy template source files to a temporary staging area for processing in %post
     install -m 0644 packaging/observability/grafana.ini.template %{buildroot}/opt/flightctl-observability/templates/
     install -m 0644 packaging/observability/flightctl-grafana.container.template %{buildroot}/opt/flightctl-observability/templates/
     
     # Copy non-templated Grafana datasource provisioning file
     install -m 0644 packaging/observability/grafana-datasources.yaml %{buildroot}/etc/grafana/provisioning/datasources/prometheus.yaml
     
     # Copy the reloader script and its systemd units
     install -m 0755 packaging/observability/flightctl-observability-config-reloader.sh %{buildroot}/usr/local/bin/
     install -m 0644 packaging/observability/flightctl-observability-config-reloader.service %{buildroot}/usr/lib/systemd/system/

%check
    %{buildroot}%{_bindir}/flightctl-agent version


%pre selinux
%selinux_relabel_pre -s %{selinuxtype}

%post selinux

%selinux_modules_install -s %{selinuxtype} %{_datadir}/selinux/packages/%{selinuxtype}/flightctl_agent.pp.bz2
%agent_relabel_files

%postun selinux

if [ $1 -eq 0 ]; then
    %selinux_modules_uninstall -s %{selinuxtype} flightctl_agent
fi

%posttrans selinux

%selinux_relabel_post -s %{selinuxtype}

# File listings
# No %files section for the main package, so it won't be built

%files cli -f licenses.list
    %{_bindir}/flightctl
    %license LICENSE
    %{_datadir}/bash-completion/completions/flightctl-completion.bash
    %{_datadir}/fish/vendor_completions.d/flightctl-completion.fish
    %{_datadir}/zsh/site-functions/_flightctl-completion

%files agent -f licenses.list
    %license LICENSE
    %dir /etc/flightctl
    %{_bindir}/flightctl-agent
    %{_bindir}/flightctl-must-gather
    /usr/lib/flightctl/hooks.d/afterupdating/00-default.yaml
    /usr/lib/systemd/system/flightctl-agent.service
    %{_sharedstatedir}/flightctl
    /usr/lib/greenboot/check/required.d/20_check_flightctl_agent.sh
    %{_docdir}/%{NAME}/*
    %{_docdir}/%{NAME}/.markdownlint-cli2.yaml
    /usr/share/sosreport/flightctl.py

%post agent
INSTALL_DIR="/usr/lib/python$(python3 --version | sed 's/^.* \(3[.][0-9]*\).*$/\1/')/site-packages/sos/report/plugins"
mkdir -p $INSTALL_DIR
cp /usr/share/sosreport/flightctl.py $INSTALL_DIR
chmod 0644 $INSTALL_DIR/flightctl.py
rm -rf /usr/share/sosreport


%files selinux
%{_datadir}/selinux/packages/%{selinuxtype}/flightctl_agent.pp.bz2

%files services
    %defattr(0644,root,root,-)
    # Files mounted to system config
    %dir %{_sysconfdir}/flightctl
    %dir %{_sysconfdir}/flightctl/pki
    %dir %{_sysconfdir}/flightctl/flightctl-api
    %dir %{_sysconfdir}/flightctl/flightctl-ui
    %dir %{_sysconfdir}/flightctl/flightctl-cli-artifacts
    %config(noreplace) %{_sysconfdir}/flightctl/service-config.yaml

    # Files mounted to data dir
    %dir %attr(0444,root,root) %{_datadir}/flightctl
    %dir %attr(0444,root,root) %{_datadir}/flightctl/flightctl-api
    %dir %attr(0444,root,root) %{_datadir}/flightctl/flightctl-db
    %dir %attr(0444,root,root) %{_datadir}/flightctl/flightctl-kv
    %dir %attr(0444,root,root) %{_datadir}/flightctl/flightctl-ui
    %dir %attr(0444,root,root) %{_datadir}/flightctl/flightctl-cli-artifacts
    %{_datadir}/flightctl/flightctl-api/config.yaml.template
    %{_datadir}/flightctl/flightctl-api/env.template
    %attr(0755,root,root) %{_datadir}/flightctl/flightctl-api/init.sh
    %attr(0755,root,root) %{_datadir}/flightctl/flightctl-api/create_aap_application.sh
    %attr(0755,root,root) %{_datadir}/flightctl/flightctl-db/enable-superuser.sh
    %{_datadir}/flightctl/flightctl-kv/redis.conf
    %{_datadir}/flightctl/flightctl-ui/env.template
    %attr(0755,root,root) %{_datadir}/flightctl/flightctl-ui/init.sh
    %attr(0755,root,root) %{_datadir}/flightctl/init_utils.sh
    %{_datadir}/flightctl/flightctl-cli-artifacts/env.template
    %{_datadir}/flightctl/flightctl-cli-artifacts/nginx.conf
    %attr(0755,root,root) %{_datadir}/flightctl/flightctl-cli-artifacts/init.sh
    %{_datadir}/containers/systemd/flightctl*

    # Handle permissions for scripts setting host config
    %attr(0755,root,root) %{_datadir}/flightctl/init_host.sh
    %attr(0755,root,root) %{_datadir}/flightctl/secrets.sh

    # Files mounted to lib dir
    /usr/lib/systemd/system/flightctl.target

%changelog

* Tue Apr 15 2025 Dakota Crowder <dcrowder@redhat.com> - 0.6.0-4
- Add ability to create an AAP Oauth Application within flightctl-services sub-package
* Fri Apr 11 2025 Dakota Crowder <dcrowder@redhat.com> - 0.6.0-3
- Add versioning to container images within flightctl-services sub-package
* Thu Apr 3 2025 Ori Amizur <oamizur@redhat.com> - 0.6.0-2
- Add sos report plugin support
* Mon Mar 31 2025 Dakota Crowder <dcrowder@redhat.com> - 0.6.0-1
- Add services sub-package for installation of containerized flightctl services
* Fri Feb 7 2025 Miguel Angel Ajo <majopela@redhat.com> - 0.4.0-1
- Add selinux support for console pty access
* Mon Nov 4 2024 Miguel Angel Ajo <majopela@redhat.com> - 0.3.0-1
- Move the Release field to -1 so we avoid auto generating packages
  with -5 all the time.
* Wed Aug 21 2024 Sam Batschelet <sbatsche@redhat.com> - 0.0.1-5
- Add must-gather script to provide a simple mechanism to collect agent debug
* Wed Aug 7 2024 Sam Batschelet <sbatsche@redhat.com> - 0.0.1-4
- Add basic greenboot support for failed flightctl-agent service
* Wed Mar 13 2024 Ricardo Noriega <rnoriega@redhat.com> - 0.0.1-3
- New specfile for both CLI and agent packages
