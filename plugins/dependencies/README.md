# Third-party dependencies

Place vendor runtime libraries here, separate from custom driver plugins.

```text
plugins/
  dependencies/
    focas/
      libfocas32.so.1
  <custom-driver>/
    linux-amd64/
      <custom-driver>.so
```

Custom driver plugins stay under `plugins/<plugin-name>/`; third-party native
libraries belong under `plugins/dependencies/<vendor>/`.

Binary files in this directory are runtime artifacts and are not committed.
