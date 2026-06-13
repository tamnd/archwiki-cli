---
title: "Quick start"
description: "Run your first archwiki command."
weight: 30
---

Once `archwiki` is on your `PATH`:

```bash
archwiki --help       # see the command tree
archwiki version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `archwiki` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
