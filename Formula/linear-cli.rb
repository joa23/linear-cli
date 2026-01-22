# typed: false
# frozen_string_literal: true

class LinearCli < Formula
  desc "Token-efficient CLI for Linear"
  homepage "https://github.com/joa23/linear-cli"
  version "1.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-darwin-arm64.tar.gz"
      sha256 "1e1a783f7e9c346f85a047ab8e6d2ef5adf858c7b8097a3530f04b6d8c92f8ef"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-darwin-amd64.tar.gz"
      sha256 "495f300ef900ee8fff635b8749866acfa1eb3c017e43ba8544cdee9de79d3a4a"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-linux-arm64.tar.gz"
      sha256 "216d9ed441270e56848d157a0e0e8158c51b3a20d2bac4ca25aef5378276e9b2"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-linux-amd64.tar.gz"
      sha256 "ce13329b925de83dae4ccb99a291b01ce1e21b8d8032152485841d9137a448dc"
    end
  end

  # This is a binary distribution - no build step required
  def pour_bottle?
    true
  end

  def install
    bin.install "linear"
  end

  test do
    assert_match "Linear CLI", shell_output("#{bin}/linear --help")
  end
end
