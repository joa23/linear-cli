# typed: false
# frozen_string_literal: true

class LinearCli < Formula
  desc "Token-efficient CLI for Linear"
  homepage "https://github.com/joa23/linear-cli"
  version "1.0.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-darwin-arm64.tar.gz"
      sha256 "9440cb0be23b5b54f75fefb5e5aa21501c19e19a58ca3c5d5e44eeb6aca48813"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-darwin-amd64.tar.gz"
      sha256 "4875f2af7a868f3178d94e8157a204a0522e7e2008db8150ba291c240f93df73"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-linux-arm64.tar.gz"
      sha256 "ed22f90d103570310b898f926df2438f6b1b070e8f14d3d6f0499bb3c8d2c041"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-linux-amd64.tar.gz"
      sha256 "ef32a2d3c93baa6e928fd7f594f741f139b4d05bc0e324e66407875b36925f95"
    end
  end

  def install
    bin.install "linear"
  end

  test do
    assert_match "Linear CLI", shell_output("#{bin}/linear --help")
  end
end
