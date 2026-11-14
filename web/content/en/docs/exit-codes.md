---
title: Exit codes
seoTitle: "qadg exit codes"
description: "What each exit code of qadg means, from 0 for a run that succeeded to 4 when your application did not do what the manifest says, as qadg --help prints them."
lead: "Every ending has its own code, so a pipeline can tell a failure of the tool from a failure of your application."
---

{{< exit-codes >}}

`130` and `143` mean the run was stopped, by you or by the system.

Only `4` is about your application: the data was produced, and your application did not do with it what the manifest says. Every other code but `0` is about the run itself, and the message on standard error says what went wrong and what to do about it.
