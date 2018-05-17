Name:           vsmtop
Version:        1.3.8
Release:        1
Summary:        VSM console performance display
Group:			Applications/Archiving
License:        AGPLv3
URL:            https://github.com/benmcclelland/vsmtop
Source0:        vsmtop-%{version}-%{release}.tar.gz

ExclusiveArch:  x86_64
# If go_compiler is not set to 1, there is no virtual provide. Use golang instead.
BuildRequires:  %{?go_compiler:compiler(go-compiler)}%{!?go_compiler:golang}

%description
%{summary}
The vsmtop utility is a top-like display for monitoring performance of VSM
processes.

%prep

%setup -n src/github.com/benmcclelland/%{name}

%build
export GOPATH=%{_builddir}
go build

%install
install -d %{buildroot}%{_bindir}
install -p -m 0755 %{name} %{buildroot}%{_bindir}/%{name}

%files
%defattr(-,root,root,-)
%attr(755,root,root) %{_bindir}/%{name}

%changelog
* Wed May 16 2018 Ben McClelland <ben.mcclelland@versity.com> - 1.3.6-1
- initial release