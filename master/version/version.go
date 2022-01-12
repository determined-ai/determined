package version

// Unset denotes that the version has not been set by the build system.
const Unset = "unknown"

// Version stores the current Determined version number when available, and unknown if otherwise not
// found. This value is set via a linker flag at build time.
// WARN: if you move it to a different package, you need to change the linked
// path in the make file and CI.
var Version = Unset
