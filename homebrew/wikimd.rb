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
      sha256 "98ead2591f4f9681d0520e12d2be3a7518488fde86c7643a2ca77d44e48e9805"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Darwin_x86_64.tar.gz"
      sha256 "7ab2a05e92c5df4cd4fe8499324e13df0717ff93ac9b07a5f7b5cc9700a8d415"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_arm64.tar.gz"
      sha256 "6b4c22441cc1ec7aa42e897d52747b1b1de919a5a57cd335df520f2395924912"
    else
      url "https://github.com/euforicio/wikimd/releases/download/v#{version}/wikimd_#{version}_Linux_x86_64.tar.gz"
      sha256 "2863beb2f651cd9d70c063d6b2c559718c291662fd865aa31a58c0fd5e7704e9"
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
