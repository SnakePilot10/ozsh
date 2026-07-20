class Ozsh < Formula
  desc "Declarative prompt builder for Zsh"
  homepage "https://github.com/snakepilot10/ozsh"
  url "https://github.com/snakepilot10/ozsh/archive/refs/tags/v0.0.0.tar.gz"
  version "0.0.0"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-ldflags", "-s -w -X main.version=#{version}", "-o", bin/"ozsh", "./cmd/ozsh"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ozsh version")
  end
end
