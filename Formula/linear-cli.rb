# typed: false
# frozen_string_literal: true

class LinearCli < Formula
  desc "Token-efficient CLI for Linear"
  homepage "https://github.com/joa23/linear-cli"
  version "1.4.7"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Darwin_arm64.tar.gz"
      sha256 "3663ef2825666e1a03df0f9b329e197e4edb22173864ffaa1433698e1d0720f1"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Darwin_x86_64.tar.gz"
      sha256 "3ed30e89d8365f053d29e61a441b7c828e7482bfb33cc435ab1570acb70f6d87"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Linux_arm64.tar.gz"
      sha256 "0740a1d4ce26dfe0c394a0cf76736c7a37cea828a792a70caf7a950ccf72f427"
    end
    on_intel do
      url "https://github.com/joa23/linear-cli/releases/download/v#{version}/linear-cli_Linux_x86_64.tar.gz"
      sha256 "c8c53766483e9d752a499907b0ea28d629fe6b55cb51464fd147fb31638b4ea2"
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
