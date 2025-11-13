# Homebrew Formula for wikimd
# Documentation: https://docs.brew.sh/Formula-Cookbook

class Wikimd < Formula
  desc "Local-first Markdown wiki with live file watching and elegant UI"
  homepage "https://github.com/euforicio/wikimd"
  version "0.1.1" # UPDATE THIS with each release
  license "MIT"

  # Platform-specific binary downloads
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Darwin_arm64.tar.gz"
      sha256 "f411305b7064711b7bfcc092aafb9f8f5332eff99659ae82f4061260f238d60f"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Darwin_x86_64.tar.gz"
      sha256 "5b23438fc50f9d009063b11e8c4a9e088544cb4122daa2fa101948b4dd2fe7af"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_arm64.tar.gz"
      sha256 "3adf655bf9a34277394934be55d833df26a349678192dae8ec5461e482e98a4b"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_x86_64.tar.gz"
      sha256 "3bbae628580271d387ea8bd3d85dce292d74c2e0749ba0d31ac007c347371b92"
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
