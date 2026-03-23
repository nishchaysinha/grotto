# typed: false
# frozen_string_literal: true

# Homebrew formula for grotto
# This file is auto-updated by GoReleaser on release
# Manual installation: brew install nishchaysinha/tap/grotto

class Grotto < Formula
  desc "A modern terminal-based code editor"
  homepage "https://github.com/nishchaysinha/grotto"
  license "GPL-3.0"
  version "0.0.0"

  on_macos do
    on_intel do
      url "https://github.com/nishchaysinha/grotto/releases/download/v#{version}/grotto-#{version}-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    end
    on_arm do
      url "https://github.com/nishchaysinha/grotto/releases/download/v#{version}/grotto-#{version}-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/nishchaysinha/grotto/releases/download/v#{version}/grotto-#{version}-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
    on_arm do
      url "https://github.com/nishchaysinha/grotto/releases/download/v#{version}/grotto-#{version}-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
  end

  def install
    bin.install "grotto"
  end

  test do
    assert_match "grotto #{version}", shell_output("#{bin}/grotto --version")
  end
end
