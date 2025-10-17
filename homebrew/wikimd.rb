# Homebrew Formula for wikimd
# Documentation: https://docs.brew.sh/Formula-Cookbook

class Wikimd < Formula
  desc "Local-first Markdown wiki with live file watching and elegant UI"
  homepage "https://github.com/euforicio/wikimd"
  version "0.1.0" # UPDATE THIS with each release
  license "MIT"

  # Platform-specific binary downloads
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Darwin_arm64.tar.gz"
      sha256 "dd832269666a8011b01d3d360ed50f3a5fd9985d90e3d974a4506b0d66832d23"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Darwin_x86_64.tar.gz"
      sha256 "fc1b96769e17153cb308fa7da38a3502a67fefac578f133b3caf9211319ca67a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_arm64.tar.gz"
      sha256 "1dbfeb3c5aeab0cdbfd98dbc89a299b76ccbd03b35a8f5c2fa79fa3ff2d48f22"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_x86_64.tar.gz"
      sha256 "35451c4d61b5ad5df2f1c16a6f97de309535711bca806c3466637f489299fa66"
    end
  end

  # Dependencies
  depends_on "ripgrep" # Required for search functionality
  uses_from_macos "curl"

  def install
    bin.install "wikimd"
  end

  test do
    # Test that binary runs and reports version
    assert_match version.to_s, shell_output("#{bin}/wikimd --version")

    # Test basic functionality
    system "#{bin}/wikimd", "--help"
  end
end
