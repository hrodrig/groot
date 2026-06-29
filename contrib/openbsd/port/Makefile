# OpenBSD port for groot (Kubernetes diagnostics CLI)
# Copy to /usr/ports/sysutils/groot/ and run: make makesum
# Submit: send diff to ports@openbsd.org

COMMENT =	read-only log and context collector for Kubernetes
DISTNAME =	groot_v0.9.0_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}
PKGNAME =	groot-0.9.0
CATEGORIES =	sysutils
HOMEPAGE =	https://github.com/hrodrig/groot
MAINTAINER =	hrodrig@usb.ve

# MIT
PERMIT_PACKAGE =	Yes
PERMIT_DISTFILES =	Yes

# Local test before release: copy tarball to DISTDIR or use
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local tarball".
MASTER_SITES =	https://github.com/hrodrig/groot/releases/download/v0.9.0/
DISTFILES =	groot_v0.9.0_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX =	.tar.gz

ONLY_FOR_ARCHS =	amd64 arm64 aarch64
NO_BUILD =	Yes
NO_TEST =	Yes
PKG_ARCH =	*

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/groot ${PREFIX}/bin/groot
	${INSTALL_DATA_DIR} ${PREFIX}/share/doc/groot
	${INSTALL_DATA} ${WRKDIR}/share/doc/groot/LICENSE ${PREFIX}/share/doc/groot/
	${INSTALL_DATA_DIR} ${PREFIX}/share/examples/groot
	${INSTALL_DATA} ${WRKDIR}/share/examples/groot/groot.yml.sample \
		${PREFIX}/share/examples/groot/

.include <bsd.port.mk>
