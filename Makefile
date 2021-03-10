PREFIX=/usr
PREFIXETC=/etc

LIB=lib
DEEPIN=deepin
PROXYFILE=deepin-proxy
DAEMON=deepin-daemon

GOPATH_DIR=gopath
GOPKG_PREFIX=github.com/DeepinProxy

GOBUILD = go build $(GO_BUILD_FLAGS)

all: build

prepare:
	mkdir -p bin
	if [ ! -d ${GOPATH_DIR}/src/$(dir ${GOPKG_PREFIX}) ];then \
	mkdir -p ${GOPATH_DIR}/src/$(dir ${GOPKG_PREFIX}); \
	ln -sf ../../.. ${GOPATH_DIR}/src/${GOPKG_PREFIX}; \
	fi

Out/%:  prepare
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o bin/${@F} ${GOPKG_PREFIX}/Out/${@F}

install:
	mkdir -p ${PREFIXETC}/${DEEPIN}/${PROXYFILE}
	install -v -D -m644 -t ${DESTDIR}${PREFIXETC}/${DEEPIN}/${PROXYFILE} misc/script/clean_script.sh
	install -v -D -m644 -t ${DESTDIR}${PREFIXETC}/${DEEPIN}/${PROXYFILE} misc/proxy/proxy.yaml
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/share/dbus-1/system.d misc/procs/com.deepin.system.procs.conf
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/share/dbus-1/system.d misc/proxy/com.deepin.system.proxy.conf
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/share/dbus-1/system-services misc/procs/com.deepin.system.procs.service
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/share/dbus-1/system-services misc/proxy/com.deepin.system.proxy.service
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/${LIB}/${DAEMON} bin/netlink
	install -v -D -m644 -t ${DESTDIR}${PREFIX}/${LIB}/${DAEMON} bin/dde-proxy


clean:
	-rm -rf bin


build: prepare Out/netlink Out/dde-proxy
