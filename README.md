# `link-checker` ![Go](https://github.com/koba-e964/link-checker/actions/workflows/go.yml/badge.svg?query=branch%3Amain)
`link-checker` is a tool that looks in a repository and ensures all HTTP links in it are alive.

# Prerequisites
- `git` should be installed
- the current directory should be managed by `git`
- Go >= 1.16 is required

# How to install
Via HomeBrew:
```bash
brew install koba-e964/tap/link-checker
```

From source:
```bash
go install github.com/koba-e964/link-checker@latest
```

# How to run
In the target directory, run:
```bash
link-checker
```

# Configuration
The configuration file is always placed in `check_links_config.toml` in the project root.

```toml
# how many times link-checker retries before giving up
retry_count = 5
# specifies files which link-checker searches for links 
text_file_extensions = [
    ".c",
    ".cpp",
    ".go",
    ".h",
    ".java",
    ".mod",
    ".md",
    ".py",
    ".rs",
    ".sh",
    ".txt",
]
```

Sometimes you may have to have links that are unstable (e.g., sometimes returns 4xx or 5xx). To handle this issue, `link-checker` allows you to have some exceptions in checking.

```toml
[[ignores]]
url = "https://csrc.nist.gov/pubs/fips/186-4/final"
codes = [200, 404] # allowed codes
reason = """
This URL seems to sometimes return 404 to requests from GitHub Actions' runners,
and the issue cannot be handled with retries."""
# considered_alternatives cannot be empty
considered_alternatives = [
    "https://www.omgwiki.org/dido/doku.php?id=dido:public:ra:xapend:xapend.b_stds:tech:nist:dss", # as flaky as the original
]
```

You can also ignore all URLs that start with a specific prefix:

```toml
[[prefix_ignores]]
prefix = "https://x.com/"
reason = "X.com links are intentionally ignored"

[[prefix_ignores]]
prefix = "https://internal.company.com/"
reason = "Internal company links are not accessible from CI"
```

# Dependency graph
![dependency graph](./dependency_graph.png)
