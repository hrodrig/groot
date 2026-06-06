# New ports collection makefile for: groot
# Date created: 2026-06-06
# Whom:        hrodrig <hrodrig@example.invalid>
#
# $FreeBSD$
#
# This is a Go-module port (FreeBSD ports collection "go-module" category).
# It builds `groot` from upstream source and installs the binary plus the
# sample config and docs.
#
# Tested on FreeBSD 14.x with go 1.26.4 (lang/go). The port follows the
# patterns established by the FreeBSD Go team for module-based ports.
# See Mk/bsd.port.mk and the ports collection's go-module examples.

PORTNAME=	groot
PORTVERSION=	0.6.0
DISTVERSIONPREFIX=	v
CATEGORIES=	sysutils

MAINTAINER=	hrodrig@users.noreply.github.com
COMMENT=	Read-only log and context collector for Kubernetes
WWW=		https://github.com/hrodrig/groot

LICENSE=	MIT
LICENSE_FILE=	${WRKSRC}/LICENSE

USES=		go:modules,1.26
USE_GITHUB=	yes
GH_ACCOUNT=	hrodrig
GH_PROJECT=	groot
GH_TAGNAME=	${DISTVERSIONPREFIX}${PORTVERSION}

# Build metadata injected the same way as GoReleaser / `make build`
GO_MODULE=	github.com/hrodrig/groot
GO_BUILDFLAGS=	-ldflags "-s -w \
			-X main.version=${PORTVERSION} \
			-X main.commit=${GH_TAGNAME} \
			-X main.branch=main \
			-X main.buildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GO_PKGNAME=	cmd/groot
GO_TARGET=	./cmd/groot

# Where the sample config and docs land; mirrors nfpms layout
GROOT_ETCDIR=	${PREFIX}/etc/groot
GROOT_DOCDIR=	${DOCSDIR}

PLIST_FILES=	bin/groot \
		etc/groot/groot.yml.sample \
		share/licenses/${PORTNAME}-${PORTVERSION}/LICENSE \
		share/doc/${PORTNAME}/README.md
PORTDOCS=	README.md CHANGELOG.md

# rc.d template is installed via the FILESDIR; see files/groot.in below.
USE_RC_SUBR=	groot

post-install:
	@${MKDIR} ${STAGEDIR}${GROOT_ETCDIR}
	${INSTALL_DATA} ${WRKSRC}/configs/groot.yml.sample \
		${STAGEDIR}${GROOT_ETCDIR}/groot.yml.sample
	@${MKDIR} ${STAGEDIR}${GROOT_DOCDIR}
	${INSTALL_DATA} ${WRKSRC}/README.md ${STAGEDIR}${GROOT_DOCDIR}/README.md
	@${MKDIR} ${STAGEDIR}${PREFIX}/share/licenses/${PORTNAME}-${PORTVERSION}
	${INSTALL_DATA} ${WRKSRC}/LICENSE \
		${STAGEDIR}${PREFIX}/share/licenses/${PORTNAME}-${PORTVERSION}/LICENSE

.include <bsd.port.mk>
