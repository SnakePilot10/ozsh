class Ozsh < Formula
  desc "Declarative prompt builder for Zsh"
  homepage "https://github.com/snakepilot10/ozsh"
  url "https://github.com/snakepilot10/ozsh/archive/refs/tags/v0.0.0.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"ozsh", "./cmd/ozsh"
  end

  test do
    system "#{bin}/ozsh"
  end
end

