unamearch=$(uname -m)
case ${unamearch} in
  x86_64 | amd64) ARCH="amd64";
  ;;
  aarch64 | arm64) ARCH="arm64";
  ;;
  *) printf "${RED}Sorry, your machine architecture ${unamearch} is not currently supported.\n${RESET}" && exit 106
  ;;
esac

echo ${ARCH}
