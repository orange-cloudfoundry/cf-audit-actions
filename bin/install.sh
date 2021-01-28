#!/usr/bin/env bash
NAME="cf-audit-actions"
REPO_NAME="cf-audit-actions"
OS=""
OWNER="orange-cloudfoundry"
: "${TMPDIR:=${TMP:-$(CDPATH=/var:/; cd -P tmp)}}"
cd -- "${TMPDIR:?NO TEMP DIRECTORY FOUND!}" || exit
cd -
echo "Installing ${NAME}..."
if [[ "$OSTYPE" == "linux-gnu" || "$(uname -s)" == "Linux" ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
elif [[ "$OSTYPE" == "cygwin" ]]; then
    OS="windows"
elif [[ "$OSTYPE" == "msys" ]]; then
    OS="windows"
elif [[ "$OSTYPE" == "win32" ]]; then
    OS="windows"
else
    echo "Os not supported by install script"
    exit 1
fi

if [[ "x$YINT_VERSION" == "x" ]]; then
    VERSION=$(curl -s https://api.github.com/repos/${OWNER}/${REPO_NAME}/releases/latest | grep tag_name | head -n 1 | cut -d '"' -f 4)
else
    VERSION=$YINT_VERSION
fi
ARCHNUM=`getconf LONG_BIT`
ARCH=""
CPUINFO=`uname -m`
VERSION_NUM=${VERSION##v}

if [[ "$ARCHNUM" == "32" ]]; then
    ARCH="386"
else
    ARCH="amd64"
fi
if [[ "$CPUINFO" == "arm"* ]]; then
    ARCH="arm"
fi
FILENAME="${NAME}_${VERSION_NUM}_${OS}_${ARCH}"
if [[ "$OS" == "windows" ]]; then
    FILENAME="${FILENAME}.exe"
fi
LINK="https://github.com/${OWNER}/${NAME}/releases/download/${VERSION}/${FILENAME}.tar.gz"
if [[ "$OS" == "windows" ]]; then
    FILEOUTPUT="${FILENAME}.tar.gz"
else
    FILEOUTPUT="${TMPDIR}/${FILENAME}.tar.gz"
fi
RESPONSE=200
if hash curl 2>/dev/null; then
    RESPONSE=$(curl --write-out %{http_code} -L -o "${FILEOUTPUT}" "$LINK")
else
    wget -o "${FILEOUTPUT}" "$LINK"
    RESPONSE=$?
fi

if [ "$RESPONSE" != "200" ] && [ "$RESPONSE" != "0" ]; then
    echo "File ${LINK} not found, so it can't be downloaded."
    rm "$FILEOUTPUT"
    exit 1
fi

tar xzf ${FILEOUTPUT} --strip 1 ${FILENAME}/${NAME}

chmod +x "${NAME}"
if [[ "$OS" == "windows" ]]; then
    mv "cf-audit-actions" "${NAME}"
else
    mv "cf-audit-actions" "/usr/local/bin/${NAME}"
fi



echo "${NAME} has been installed."
