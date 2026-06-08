PORTNAME=	groot
PORTVERSION=	0.6.0
DISTVERSIONPREFIX=	v
CATEGORIES=	sysutils
# Local test before release: copy the tarball into DISTDIR (see `make -V DISTDIR`
# from this port directory) using the same name as DISTFILES, or run e.g.
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local distfile".
MASTER_SITES=	https://github.com/hrodrig/groot/releases/download/v${PORTVERSION}/
DISTFILES=	groot_v${PORTVERSION}_freebsd_${ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX=	.tar.gz

MAINTAINER=	hrodrig@usb.ve
COMMENT=	Read-only log and context collector for Kubernetes
WWW=		https://github.com/hrodrig/groot
LICENSE=	MIT

NO_BUILD=	yes
ONLY_FOR_ARCHS=	amd64 aarch64

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/groot ${STAGEDIR}${PREFIX}/bin/groot
	${MKDIR} ${STAGEDIR}${PREFIX}/share/doc/groot
	${INSTALL_DATA} ${WRKDIR}/share/doc/groot/LICENSE ${STAGEDIR}${PREFIX}/share/doc/groot/
	${MKDIR} ${STAGEDIR}${PREFIX}/share/examples/groot
	${INSTALL_DATA} ${WRKDIR}/share/examples/groot/groot.yml.sample \
		${STAGEDIR}${PREFIX}/share/examples/groot/

.include <bsd.port.mk>
