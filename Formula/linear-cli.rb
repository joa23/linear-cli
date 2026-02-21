# typed: false
# frozen_string_literal: true

class LinearCli < Formula
  desc "Token-efficient CLI for Linear"
  homepage "https://github.com/joa23/linear-cli"
  version "1.5.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Darwin_arm64.tar.gz"
      sha256 "324bd53686445742ed04aa961dbcb6e9bc1206ef00afff51c9c1c2a2aad62a3c"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Darwin_x86_64.tar.gz"
      sha256 "0227b96c1a1fc8edcdd110c189b87c10579abe90a90c3d7b38b51096d1d089d2"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Linux_arm64.tar.gz"
      sha256 "1fae10d968a4c13c3d72d2e27fd77193e6d57457434cd59ccf735db317d7ec6b"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Linux_x86_64.tar.gz"
      sha256 "70002589c1567e6a9ecd95d4f53165e07217473f44f2164d32e1edd60c5aab1e"
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
