# FOCAS runtime library

Place the FANUC FOCAS shared library in this directory before enabling the
`CNC / fanuc focas` driver.

Expected names (first match wins):

- `libfocas32.so.1`
- `libfocas32.so`
- `libfwlib32-linux-x64.so.1.0.5`
- `libfwlib32.so.1`

The path can also be overridden with `FOCAS_LIBRARY_PATH`.

The gateway currently only provides the FOCAS driver framework. The cgo
adapter for `cnc_allclibhndl3` / `pmc_rdpmcrng` / `cnc_rdposition` still needs
to be wired to the vendor library and its matching headers.
