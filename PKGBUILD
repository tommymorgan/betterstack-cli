# Maintainer: Tommy Morgan <tommy@tommymorgan.com>
pkgname=betterstack-cli-git
pkgver=r13.aa53b47
pkgrel=1
pkgdesc='CLI for interacting with the BetterStack API'
arch=('x86_64' 'aarch64')
url='https://github.com/tommymorgan/betterstack-cli'
license=('UNLICENSED')
makedepends=('go')
provides=('betterstack-cli')
conflicts=('betterstack-cli')
source=("${pkgname}::git+${url}.git")
sha256sums=('SKIP')

pkgver() {
  cd "${pkgname}"
  printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

build() {
  cd "${pkgname}"
  export CGO_ENABLED=0
  go build -trimpath -buildmode=pie \
    -ldflags "-s -w -X github.com/tommymorgan/betterstack-cli/cmd/betterstack.Version=${pkgver}" \
    -o betterstack-cli .
}

package() {
  cd "${pkgname}"
  install -Dm755 betterstack-cli "${pkgdir}/usr/bin/betterstack-cli"
}
