Resolve each path to its canonical absolute form.

    PATH [PATH...]

Resolves symlinks and removes . and .. components.

Examples:
    ../config.yaml      → /home/user/project/config.yaml
    ./symlink           → /actual/target/path
