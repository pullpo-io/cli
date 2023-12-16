# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Pullpo < Formula
  desc "Control Pullpo, GitHub and GitLab from the terminal."
  homepage "https://pullpo.io/"
  version "0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/pullpo-io/cli/releases/download/v0.3/pullpo_0.3_macOS_arm64.zip"
      sha256 "138db9806fe06a8ea39d8adbf2d8678be58f54acb60c79820395b01c799d36dd"

      def install
        bin.install "bin/pullpo"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/pullpo-io/cli/releases/download/v0.3/pullpo_0.3_macOS_amd64.zip"
      sha256 "db2dabf664e63757c34356a0096d529c37700dfbf083ceba20a410727364e405"

      def install
        bin.install "bin/pullpo"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && !Hardware::CPU.is_64_bit?
      url "https://github.com/pullpo-io/cli/releases/download/v0.3/pullpo_0.3_linux_armv6.tar.gz"
      sha256 "f928fa0bc0a4b0ec8c12fe13f4248ee186e27c13f3690e6bbbab41caa3211fb5"

      def install
        bin.install "bin/pullpo"
      end
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/pullpo-io/cli/releases/download/v0.3/pullpo_0.3_linux_arm64.tar.gz"
      sha256 "1f5697685e1fb8b6f05d8e30e096c69da468eed1509d34ceb04b90ffbc008a41"

      def install
        bin.install "bin/pullpo"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/pullpo-io/cli/releases/download/v0.3/pullpo_0.3_linux_amd64.tar.gz"
      sha256 "19a26a2edd7890a18d8a208bfad0d6e2840a4828170fdcb6990e172fe7f6daa1"

      def install
        bin.install "bin/pullpo"
      end
    end
  end
end
