# holen - Utility fetcher

# Quick Start

1. [Download the holen binary](https://github.com/justone/holen/releases) (at least v0.3.0-pre1 or above) for your platform and place it in your \$PATH.
2. Run `holen source update` to fetch this main repo of manifests.
3. Run `holen list` to view the available commands.
4. Run `holen link [utility] -b [dir to link into]` to link a utility.  For example, `holen link jq -b $HOME/bin`.
5. Run the utility.

# Selecting a different strategy:

By default, holen tries strategies in the following order:

1. Docker
2. Binary

If you'd like it to try binary first, just run `holen config strategy.priority binary,docker`.
