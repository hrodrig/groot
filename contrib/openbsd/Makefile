# $OpenBSD$
#
# OpenBSD port Makefile for groot. Drop into /usr/ports/sysutils/groot/
# (or use contrib/openbsd/ as the source tree) and `make package` on
# OpenBSD 7.5+ with lang/go 1.26 installed.
#
# Follows the conventions in https://openbsd-ports.github.io/ for Go-module
# ports (MODGO pattern). Verified against kzero's contrib/openbsd layout
# with adjustments for GROOT's module path and build tags.

COMMENT=		read-only log and context collector for Kubernetes

GH_ACCOUNT=		hrodrig
GH_PROJECT=		groot
GH_TAGNAME=		v0.6.0

CATEGORIES=		sysutils
HOMEPAGE=		https://github.com/hrodrig/groot
MAINTAINER=		Hector Rodriguez <hrodrig@users.noreply.github.com>

# MIT is in the tree
PERMIT_PACKAGE=		Yes

MODULES=		lang/go

MODGO_MOD_NAME=		github.com/hrodrig/groot
MODGO_VERSION=		v0.6.0
MODGO_LDFLAGS=		-X main.version=0.6.0 \
			-X main.commit=v0.6.0 \
			-X main.branch=main \
			-X main.buildDate=
WANTLIB=		pthread

# Where we install the sample config; nfpms parity
EXAMPLES_DIR=		${PREFIX}/share/examples/groot

do-install:
	${INSTALL_PROGRAM_DIR} ${PREFIX}/bin
	${INSTALL_PROGRAM} -m 555 ${MODGO_WORKSPACE}/bin/groot ${PREFIX}/bin/groot
	${INSTALL_DATA_DIR} ${EXAMPLES_DIR}
	${INSTALL_DATA} ${WRKSRC}/configs/groot.yml.sample \
		${EXAMPLES_DIR}/groot.yml.sample
	${INSTALL_DATA_DIR} ${PREFIX}/share/doc/groot
	${INSTALL_DATA} ${WRKSRC}/README.md ${PREFIX}/share/doc/groot/README.md
	${INSTALL_DATA} ${WRKSRC}/CHANGELOG.md ${PREFIX}/share/doc/groot/CHANGELOG.md
	${INSTALL_DATA} ${WRKSRC}/LICENSE ${PREFIX}/share/doc/groot/LICENSE

.include <bsd.port.mk>
